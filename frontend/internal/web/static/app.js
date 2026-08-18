(() => {
  'use strict';

  const selectionKey = 'bakery:selected-orders';
  const selectionModeKey = 'bakery:selection-mode';
  const orderCooldownKey = 'bakery:last-order-create';
  const orderCooldownMs = 5000;

  document.addEventListener('htmx:configRequest', (event) => {
    const token = document.querySelector('meta[name="csrf-token"]')?.content;
    if (token) event.detail.headers['X-CSRF-Token'] = token;
  });

  document.addEventListener('htmx:beforeSwap', (event) => {
    if (event.detail.xhr.status >= 400 && event.detail.xhr.status < 600) {
      event.detail.shouldSwap = true;
      event.detail.isError = false;
    }
  });

  // htmx's own boost handler binds directly to each boosted form/link during
  // htmx.process(), which runs on the FIRST DOMContentLoaded listener (its
  // script loads before app.js). Our own per-form submit listener used to be
  // registered second on the very same element, and same-element listeners
  // fire in registration order regardless of capture — so htmx's boosted
  // request already fired before our confirm dialog ever opened, and
  // pressing "Отмена" in the dialog did nothing. Registering these gates in
  // the capture phase on `document` instead always wins the race, because
  // capture-phase listeners on an ancestor run before any listener on the
  // target itself, no matter which one was attached first.
  document.addEventListener('submit', (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement) || form.dataset.confirmed === 'true') return;
    const message = event.submitter?.dataset.confirm || form.dataset.confirm;
    if (!message) return;
    event.preventDefault();
    event.stopPropagation();
    showConfirm(message, () => {
      form.dataset.confirmed = 'true';
      form.requestSubmit(event.submitter);
    });
  }, true);

  document.addEventListener('click', (event) => {
    const link = event.target.closest('a[data-confirm]');
    if (!link || link.dataset.confirmed === 'true') return;
    event.preventDefault();
    event.stopPropagation();
    showConfirm(link.dataset.confirm, () => {
      link.dataset.confirmed = 'true';
      link.click();
    });
  }, true);

  // Block duplicate submissions. Once a form is mid-submit, swallow further
  // submit events in the capture phase — which runs before htmx's own
  // form-level boost handler — so a fast double-click can't fire a second
  // request (native or boosted). The flag clears when the request settles
  // (htmx:afterRequest) or, on success, when the page swaps in a fresh form.
  document.addEventListener('submit', (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    if (form.dataset.submitting === 'true') { event.preventDefault(); event.stopImmediatePropagation(); return; }
    if (event.defaultPrevented) return; // a confirm dialog deferred this submit
    form.dataset.submitting = 'true';
    // A create submit posts to /orders (drafts go to /orders/draft, edits to
    // /orders/{n}/edit) — remember it so the cooldown can start on success.
    const action = (event.submitter?.getAttribute('formaction') || form.getAttribute('action') || '').split('?')[0];
    form.dataset.createSubmit = /\/orders$/.test(action) ? 'true' : '';
    // Deferred so the submitter's value is still serialized into the request.
    setTimeout(() => form.querySelectorAll('button[type="submit"]').forEach((b) => { b.disabled = true; }), 0);
  }, true);

  document.addEventListener('htmx:afterRequest', (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement) || form.dataset.submitting !== 'true') return;
    // On success the page swaps in a fresh form, so stay locked until then;
    // only unlock a failed request so the user can fix and retry.
    if (event.detail?.successful) {
      if (form.dataset.createSubmit === 'true') { try { localStorage.setItem(orderCooldownKey, String(Date.now())); } catch (e) { /* storage disabled */ } }
      return;
    }
    delete form.dataset.submitting;
    form.querySelectorAll('button[type="submit"]').forEach((b) => { b.disabled = false; });
  });

  document.addEventListener('htmx:afterSwap', () => initialize(document));
  document.addEventListener('htmx:historyRestore', () => {
    // htmx's own back/forward cache swaps in a stringified DOM snapshot: listeners
    // are gone but data-ready attributes baked into that snapshot survive, so the
    // usual :not([data-ready]) guards would skip re-wiring everything silently.
    document.querySelectorAll('[data-ready]').forEach((el) => el.removeAttribute('data-ready'));
    initialize(document);
  });
  document.addEventListener('DOMContentLoaded', () => {
    initialize(document);
    initializeTelegram();
  });

  function initialize(root) {
    initializeDialogs(root);
    initializeSelection(root);
    initializeSelectionRemoval(root);
    initializeCatalog(root);
    initializeCatalogTabs(root);
    initializeSortable(root);
    initializeCommentToggles(root);
    initializeProduction(root);
    initializeMonitorScale(root);
    initializeScrollHints(root);
    initializeOrderCooldown(root);
    initializeDateChips(root);
  }

  // Quick-pick chips next to the fulfillment-date input: today, tomorrow and
  // the day after (labelled by weekday). Built client-side so "today" is the
  // user's local day, matching what the date input itself would show.
  function initializeDateChips(root) {
    root.querySelectorAll('[data-date-chips]:not([data-ready])').forEach((wrap) => {
      wrap.dataset.ready = 'true';
      const input = wrap.closest('label')?.querySelector('[data-date-input]');
      if (!input) return;
      const iso = (offset) => {
        const day = new Date();
        day.setDate(day.getDate() + offset);
        return `${day.getFullYear()}-${String(day.getMonth() + 1).padStart(2, '0')}-${String(day.getDate()).padStart(2, '0')}`;
      };
      const weekday = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'][(new Date().getDay() + 2) % 7];
      const chips = [['Сегодня', iso(0)], ['Завтра', iso(1)], [weekday, iso(2)]].map(([label, value]) => {
        const chip = document.createElement('button');
        chip.type = 'button';
        chip.className = 'date-chip';
        chip.textContent = label;
        chip.dataset.date = value;
        chip.addEventListener('click', () => {
          input.value = value;
          input.dispatchEvent(new Event('change', { bubbles: true }));
          render();
        });
        wrap.append(chip);
        return chip;
      });
      const render = () => chips.forEach((chip) => chip.setAttribute('aria-pressed', String(chip.dataset.date === input.value)));
      input.addEventListener('change', render);
      render();
    });
  }

  // Frontend rate-limit: after an order is created, keep the create button
  // disabled for 5s (persisted, so it survives the redirect to /orders and a
  // quick "back" to the form). UX guard only — the real anti-duplicate is the
  // capture-phase submit lock; this just stops rapid repeat creates.
  function initializeOrderCooldown(root) {
    const editor = root.querySelector('form.editor-form[action="/orders"]');
    const btn = editor?.querySelector('.sticky-submit .button-primary[type="submit"]');
    if (!btn || btn.dataset.cooldownReady === 'true') return;
    btn.dataset.cooldownReady = 'true';
    const label = btn.textContent;
    const tick = () => {
      let ts = 0;
      try { ts = Number(localStorage.getItem(orderCooldownKey)) || 0; } catch (e) { /* storage disabled */ }
      const remain = orderCooldownMs - (Date.now() - ts);
      if (remain > 0) {
        btn.disabled = true;
        btn.textContent = 'Подождите ' + Math.ceil(remain / 1000) + ' с';
        setTimeout(tick, 250);
      } else {
        btn.textContent = label;
        if (btn.form?.dataset.submitting !== 'true') btn.disabled = false;
      }
    };
    tick();
  }

  // Marks a scroll frame while its table still has content to the right, so the
  // edge fade only appears when dragging sideways would actually reveal something.
  function initializeScrollHints(root) {
    root.querySelectorAll('.scroll-frame:not([data-ready])').forEach((frame) => {
      frame.dataset.ready = 'true';
      const scroller = frame.firstElementChild;
      if (!scroller) return;
      const update = () => {
        const more = scroller.scrollWidth - scroller.clientWidth - scroller.scrollLeft > 2;
        frame.dataset.scrollMore = String(more);
      };
      scroller.addEventListener('scroll', update, { passive: true });
      new ResizeObserver(update).observe(scroller);
      update();
    });
  }

  function initializeSelectionRemoval(root) {
    const page = root.querySelector('[data-selection-page]');
    if (!page || page.dataset.ready === 'true') return;
    page.dataset.ready = 'true';
    const buttons = [...page.querySelectorAll('[data-selection-remove]')];
    const query = new URLSearchParams(window.location.search);
    if (buttons.length === 0 && !query.has('order') && !query.has('orders')) {
      const saved = selection();
      if (saved.length > 0) {
        window.location.replace(selectionURL(saved));
        return;
      }
    }
    const current = buttons.map((button) => ({
      number: button.dataset.selectionRemove,
      category: button.dataset.selectionCategory,
      categoryName: button.dataset.selectionCategoryName || '',
    }));
    storeSelection(current);
    storeSelectionMode(current.length > 0);
    buttons.forEach((button) => {
      button.addEventListener('click', (event) => {
        event.stopPropagation();
        const remaining = current.filter((item) => item.number !== button.dataset.selectionRemove);
        storeSelection(remaining);
        storeSelectionMode(remaining.length > 0);
        window.location.assign(remaining.length ? selectionURL(remaining) : '/orders');
      });
    });
  }

  function initializeDialogs(root) {
    root.querySelectorAll('dialog[data-auto-dialog]:not([data-ready])').forEach((dialog) => {
      dialog.dataset.ready = 'true';
      dialog.addEventListener('close', () => dialog.remove());
      dialog.addEventListener('click', (event) => { if (event.target === dialog) dialog.close(); });
      dialog.showModal();
    });
    // Manual dialogs: opened by a [data-open-dialog] button, closed on backdrop
    // click or a [data-close-dialog] button. Unlike the auto ones they persist.
    root.querySelectorAll('dialog[data-dialog]:not([data-ready])').forEach((dialog) => {
      dialog.dataset.ready = 'true';
      dialog.addEventListener('click', (event) => {
        if (event.target === dialog) dialog.close();
        if (event.target.closest('[data-close-dialog]')) dialog.close();
      });
    });
    root.querySelectorAll('[data-open-dialog]:not([data-ready])').forEach((opener) => {
      opener.dataset.ready = 'true';
      opener.addEventListener('click', () => document.getElementById(opener.dataset.openDialog)?.showModal());
    });
  }

  function initializeTelegram() {
    const telegram = window.Telegram?.WebApp;
    if (!telegram) return;
    telegram.ready();
    telegram.expand();
    // Swiping down over a scrolled table otherwise drags/collapses the mini app
    // window instead of scrolling the table (Bot API 7.7+).
    telegram.disableVerticalSwipes?.();
    telegram.setHeaderColor?.('#f4f3ee');
    telegram.setBackgroundColor?.('#f4f3ee');
    const login = document.querySelector('[data-telegram-login]');
    if (!login || !telegram.initData) return;
    // If auto-login fails (account not linked yet), the password form stays
    // usable; sending initData with it lets the backend bind telegram_id.
    login.querySelectorAll('[data-telegram-init-data]').forEach((input) => { input.value = telegram.initData; });
    const status = login.querySelector('.telegram-status');
    if (status) status.textContent = 'Подтверждаем вход через Telegram…';
    fetch('/session/telegram', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ init_data: telegram.initData, next: login.dataset.next || '/orders' }),
    }).then((response) => {
      if (!response.ok) return response.text().then((message) => Promise.reject(new Error(message)));
      window.location.assign(response.headers.get('HX-Location') || '/orders');
    }).catch((error) => {
      if (status) status.textContent = error.message || 'Не удалось войти через Telegram.';
    });
  }

  function showConfirm(message, confirm) {
    const dialog = document.createElement('dialog');
    dialog.className = 'confirm-dialog';
    dialog.innerHTML = '<form method="dialog"><div><p class="eyebrow">Подтверждение</p><h2>Продолжить?</h2></div><p></p><menu><button class="button" value="cancel" autofocus>Отмена</button><button class="button button-danger" value="confirm">Подтвердить</button></menu></form>';
    dialog.querySelector('p:last-of-type').textContent = message || 'Подтвердите действие.';
    dialog.addEventListener('close', () => {
      if (dialog.returnValue === 'confirm') confirm();
      dialog.remove();
    });
    document.body.append(dialog);
    dialog.showModal();
  }

  // Every hx-boost navigation replaces #app-shell and re-runs initializeSelection,
  // which used to re-read sessionStorage from scratch each time. In WebViews that
  // block storage writes (Telegram in-app browser, private mode, strict partition
  // policies) that re-read silently comes back empty, wiping an in-progress pick
  // on the very next navigation. The in-memory cache below is the source of truth
  // for the life of the document; sessionStorage is only a best-effort way to
  // survive an actual full reload, and a failure to persist there no longer loses
  // anything for the current tab.
  let selectionCache = null;
  let selectionModeCache = null;

  function selection() {
    if (selectionCache) return selectionCache;
    let value = [];
    try {
      const parsed = JSON.parse(sessionStorage.getItem(selectionKey) || '[]');
      if (Array.isArray(parsed)) value = parsed;
    } catch { /* fall back to an empty starting selection */ }
    const seen = new Set();
    selectionCache = value.flatMap((item) => {
      const number = String(item?.number || '').trim();
      const category = String(item?.category ?? '').trim();
      if (!number || !category || seen.has(number)) return [];
      seen.add(number);
      return [{ number, category, categoryName: String(item.categoryName || '') }];
    });
    return selectionCache;
  }

  function storeSelection(value) {
    selectionCache = value;
    try { sessionStorage.setItem(selectionKey, JSON.stringify(value)); } catch { /* still works for this tab via selectionCache */ }
  }

  function selectionURL(value) {
    const params = new URLSearchParams();
    value.forEach((item) => params.append('order', item.number));
    return `/orders/selection?${params}`;
  }

  function selectionMode() {
    if (selectionModeCache !== null) return selectionModeCache;
    try { selectionModeCache = sessionStorage.getItem(selectionModeKey) === 'true'; } catch { selectionModeCache = false; }
    return selectionModeCache;
  }

  function storeSelectionMode(value) {
    selectionModeCache = value;
    try { sessionStorage.setItem(selectionModeKey, String(value)); } catch { /* still works for this tab via selectionModeCache */ }
  }

  function initializeSelection(root) {
    const toggle = root.querySelector('[data-selection-toggle]');
    const cards = [...root.querySelectorAll('[data-select-order]')];
    if (!toggle) return;
    if (toggle.dataset.ready === 'true') return;
    toggle.dataset.ready = 'true';

    let selected = selection();
    let mode = selectionMode() || selected.length > 0;
    const page = toggle.closest('.orders-page');
    const open = root.querySelector('[data-selection-open]');
    const count = root.querySelector('[data-selection-count]');
    const blocked = (card) => card.dataset.cancelled === 'true' || Boolean(card.dataset.sheet);

    const render = () => {
      // An order produced or cancelled since it was picked cannot stay in the
      // batch: pick() refuses to add one, so it must not survive in storage
      // either. Only cards on screen can be judged; the rest the server drops.
      const stale = new Set(cards.filter(blocked).map((card) => card.dataset.number));
      if (selected.some((item) => stale.has(item.number))) {
        selected = selected.filter((item) => !stale.has(item.number));
        storeSelection(selected);
      }
      page?.classList.toggle('selection-active', mode);
      toggle.setAttribute('aria-pressed', String(mode));
      toggle.textContent = mode ? 'Отмена' : 'Выбрать заказы';
      cards.forEach((card) => {
        const picked = selected.some((item) => item.number === card.dataset.number);
        card.classList.toggle('is-selected', picked);
        // Only a card you can actually toggle should announce itself as a button.
        if (mode) {
          card.setAttribute('role', 'button');
          card.setAttribute('tabindex', '0');
          card.setAttribute('aria-pressed', String(picked));
          // aria-disabled must carry the literal "true"; an empty value reads as false.
          if (blocked(card)) card.setAttribute('aria-disabled', 'true');
          else card.removeAttribute('aria-disabled');
        } else {
          ['role', 'tabindex', 'aria-pressed', 'aria-disabled'].forEach((name) => card.removeAttribute(name));
        }
      });
      if (count) count.textContent = String(selected.length);
      if (open) {
        open.hidden = !mode || selected.length === 0;
        open.href = selectionURL(selected);
      }
    };

    const pick = (card) => {
      if (card.dataset.cancelled === 'true') {
        showToast('Отменённый заказ нельзя добавить в партию.');
        return;
      }
      if (card.dataset.sheet) {
        showToast(`Заказ уже входит в отработку №${card.dataset.sheet}.`);
        return;
      }
      const number = card.dataset.number;
      const found = selected.findIndex((item) => item.number === number);
      if (found >= 0) {
        selected.splice(found, 1);
      } else if (selected.length && selected[0].category !== card.dataset.category) {
        const locked = selected[0].categoryName || 'другого типа';
        showToast(`В партии уже заявки «${locked}». Снимите их, чтобы выбрать другой тип.`);
        return;
      } else {
        selected.push({ number, category: card.dataset.category, categoryName: card.dataset.categoryName || '' });
      }
      storeSelection(selected);
      render();
    };

    toggle.addEventListener('click', () => {
      if (mode) selected = [];
      mode = !mode;
      storeSelection(selected);
      storeSelectionMode(mode);
      render();
    });

    if (open) {
      // Continuing commits the batch — its orders travel in the link URL. Drop the
      // stored picks and mode so returning to the matrix doesn't reopen selection
      // with stale highlights.
      open.addEventListener('click', () => {
        storeSelection([]);
        storeSelectionMode(false);
      });
    }

    cards.forEach((card) => {
      // Capture phase: hx-boost binds its own handler to the card link, and on the
      // bubble path that runs before this one, so the boosted navigation would win.
      card.addEventListener('click', (event) => {
        if (event.target.closest('button, input, select, textarea')) return;
        if (!mode) return;
        event.preventDefault();
        event.stopPropagation();
        pick(card);
      }, true);
      card.addEventListener('keydown', (event) => {
        if (!mode || event.target !== card || (event.key !== 'Enter' && event.key !== ' ')) return;
        event.preventDefault();
        pick(card);
      });
    });
    render();
  }

  function initializeCatalog(root) {
    root.querySelectorAll('[data-order-editor], [data-calculator]').forEach((editor) => {
      if (editor.dataset.ready === 'true') return;
      editor.dataset.ready = 'true';
      const rows = [...editor.querySelectorAll('[data-catalog-row]')];
      const search = editor.querySelector('[data-dish-search]');
      const radios = [...editor.querySelectorAll('input[name="category_id"]')];
      const updateRows = () => {
        const category = radios.find((radio) => radio.checked)?.value || '';
        const term = (search?.value || '').trim().toLocaleLowerCase('ru');
        rows.forEach((row) => {
          const categoryMatch = category && (row.dataset.category === '0' || row.dataset.category === category);
          const searchMatch = !term || (row.dataset.name || '').toLocaleLowerCase('ru').includes(term);
          const visible = categoryMatch && searchMatch;
          row.hidden = !visible;
          row.querySelectorAll('input').forEach((input) => { input.disabled = !categoryMatch; });
        });
        editor.querySelectorAll('.catalog-group').forEach((group) => {
          group.hidden = !group.querySelector('[data-catalog-row]:not([hidden])');
        });
        editor.querySelectorAll('.category-choice').forEach((choice) => choice.classList.toggle('is-selected', Boolean(choice.querySelector('input:checked'))));
        updateSummary(editor);
      };
      radios.forEach((radio) => radio.addEventListener('change', updateRows));
      // The radio is visually hidden (opacity:0), so the browser can't anchor its
      // "please select" bubble to it — submit is blocked with no feedback and the
      // dish rows stay hidden until a type is picked. Surface the invalid state so
      // the form doesn't look broken. `invalid` still fires on the hidden control.
      let warnedAt = 0;
      radios.forEach((radio) => radio.addEventListener('invalid', () => {
        // Every radio in the group reports invalid on the same submit; toast once.
        if (Date.now() - warnedAt < 300) return;
        warnedAt = Date.now();
        showToast('Выберите тип заявки — без него не видны позиции и заказ не сохранится.');
        editor.querySelector('.category-picker')?.scrollIntoView({ block: 'center', behavior: 'smooth' });
      }));
      search?.addEventListener('input', updateRows);
      editor.querySelectorAll('.quantity-input').forEach((input) => input.addEventListener('input', () => updateSummary(editor)));
      updateRows();
    });
  }

  // Per-category tabs on the admin catalog: show one panel at a time. Only
  // buttons carrying data-tab are real tabs — the "＋ Тип" button opens a dialog.
  function initializeCatalogTabs(root) {
    root.querySelectorAll('[data-catalog-tabs]:not([data-ready])').forEach((tabs) => {
      tabs.dataset.ready = 'true';
      const buttons = [...tabs.querySelectorAll('.tab-button[data-tab]')];
      const panels = [...tabs.querySelectorAll('.tab-panel')];
      const show = (id) => {
        buttons.forEach((button) => button.setAttribute('aria-selected', String(button.dataset.tab === id)));
        panels.forEach((panel) => { panel.hidden = panel.id !== id; });
      };
      buttons.forEach((button) => button.addEventListener('click', () => show(button.dataset.tab)));
      const active = buttons.find((button) => button.getAttribute('aria-selected') === 'true') || buttons[0];
      if (active) show(active.dataset.tab);
    });
  }

  // Dish ordering via ↑/↓ buttons: move the row past its sibling, then save the
  // full catalogue order (every tab, in DOM order) so the stored sort matches what
  // the admin sees. The buttons live in the row's <summary>, so cancel the click's
  // default toggle before moving.
  function initializeSortable(root) {
    root.querySelectorAll('[data-sortable]:not([data-ready])').forEach((list) => {
      list.dataset.ready = 'true';
      list.addEventListener('click', (event) => {
        const button = event.target.closest('[data-move]');
        if (!button) return;
        event.preventDefault();
        event.stopPropagation();
        const item = button.closest('.sortable-item');
        if (!item) return;
        if (button.dataset.move === 'up') {
          const prev = item.previousElementSibling;
          if (prev && prev.classList.contains('sortable-item')) list.insertBefore(item, prev);
        } else {
          const next = item.nextElementSibling;
          if (next && next.classList.contains('sortable-item')) list.insertBefore(next, item);
        }
        saveDishOrder();
      });
    });
  }

  function saveDishOrder() {
    const codes = [...document.querySelectorAll('.sortable-item[data-dish-code]')].map((item) => item.dataset.dishCode);
    if (codes.length === 0) return;
    const body = new URLSearchParams();
    codes.forEach((code) => body.append('codes', code));
    const token = document.querySelector('meta[name="csrf-token"]')?.content || '';
    fetch('/admin/dishes/reorder', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-CSRF-Token': token },
      body,
    }).then((response) => { if (!response.ok) showToast('Не удалось сохранить порядок блюд.'); })
      .catch(() => showToast('Не удалось сохранить порядок блюд.'));
  }

  function initializeCommentToggles(root) {
    root.querySelectorAll('[data-comment-toggle]:not([data-ready])').forEach((button) => {
      button.dataset.ready = 'true';
      const label = button.closest('[data-catalog-row], [data-production-row]')?.querySelector('.item-comment');
      if (!label) return;
      button.setAttribute('aria-expanded', String(!label.hidden));
      button.addEventListener('click', () => {
        label.hidden = !label.hidden;
        button.setAttribute('aria-expanded', String(!label.hidden));
        if (!label.hidden) label.querySelector('input')?.focus();
      });
    });
  }

  function updateSummary(editor) {
    const output = editor.querySelector('[data-editor-summary]');
    if (!output) return;
    let count = 0;
    editor.querySelectorAll('[data-catalog-row]:not([hidden])').forEach((row) => {
      const total = [...row.querySelectorAll('.quantity-input')].reduce((sum, input) => sum + (Number(input.value) || 0), 0);
      if (total > 0) count += 1;
    });
    output.textContent = `${count} поз.`;
  }

  function initializeProduction(root) {
    root.querySelectorAll('[data-production-form]:not([data-ready])').forEach((form) => {
      form.dataset.ready = 'true';
      const initial = new FormData(form);
      const baseline = new URLSearchParams(initial).toString();
      const warning = document.querySelector('[data-dirty-warning]');
      const monitor = document.querySelector('[data-monitor-button]');
      const result = document.getElementById('monitor-result');
      const updateDirty = () => {
        const dirty = new URLSearchParams(new FormData(form)).toString() !== baseline;
        if (warning) warning.hidden = !dirty;
        if (monitor) monitor.disabled = dirty;
        if (result) result.hidden = dirty;
      };
      form.querySelectorAll('[data-production-row]').forEach((row) => {
        const loaded = row.querySelector('[name="loaded_quantity"]');
        const produced = row.querySelector('[name="produced_quantity"]');
        if (!loaded || !produced) return;
        produced.addEventListener('input', () => { row.dataset.linked = 'false'; updateDirty(); });
        loaded.addEventListener('input', () => {
          if (row.dataset.linked === 'true') produced.value = loaded.value;
          updateDirty();
        });
      });
      form.addEventListener('input', updateDirty);
      form.querySelector('[data-production-reset]')?.addEventListener('click', () => { form.reset(); form.querySelectorAll('[data-production-row]').forEach((row) => { row.dataset.linked = row.dataset.initialLinked || 'false'; }); updateDirty(); });
    });
  }

  function initializeMonitorScale(root) {
    root.querySelectorAll('[data-monitor-report]:not([data-ready])').forEach((report) => {
      report.dataset.ready = 'true';
      const target = report.querySelector('[data-monitor-target]');
      const base = Number(report.dataset.baseQty) || 0;
      if (!target || base <= 0) return;
      const update = () => {
        const next = Number(target.value);
        const scale = target.value === '' || target.value === target.dataset.initialValue || !Number.isFinite(next) ? 1 : next / base;
        report.querySelectorAll('[data-monitor-qty]').forEach((node) => {
          const qty = (Number(node.dataset.baseQty) || 0) * scale;
          node.textContent = `${formatMonitorQty(qty)} ${node.dataset.unit || ''}`.trim();
        });
      };
      target.addEventListener('input', update);
      update();
    });
  }

  function formatMonitorQty(value) {
    const rounded = Math.round(value * 1000) / 1000;
    return Number.isInteger(rounded) ? String(rounded) : String(rounded).replace(/0+$/, '').replace(/\.$/, '');
  }

  function showToast(message) {
    const region = document.getElementById('toast-region');
    if (!region) return;
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = message;
    region.append(toast);
    window.setTimeout(() => toast.remove(), 3500);
  }
})();
