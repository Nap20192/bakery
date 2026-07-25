(() => {
  'use strict';

  const selectionKey = 'bakery:selected-orders';
  const selectionModeKey = 'bakery:selection-mode';

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

  document.addEventListener('htmx:afterSwap', () => initialize(document));
  document.addEventListener('DOMContentLoaded', () => {
    initialize(document);
    initializeTelegram();
  });

  function initialize(root) {
    initializeConfirmations(root);
    initializeDialogs(root);
    initializeSelection(root);
    initializeSelectionRemoval(root);
    initializeCatalog(root);
    initializeCommentToggles(root);
    initializeProduction(root);
    initializeScrollHints(root);
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
  }

  function initializeTelegram() {
    const telegram = window.Telegram?.WebApp;
    if (!telegram) return;
    telegram.ready();
    telegram.expand();
    telegram.setHeaderColor?.('#f4f3ee');
    telegram.setBackgroundColor?.('#f4f3ee');
    const login = document.querySelector('[data-telegram-login]');
    if (!login || !telegram.initData) return;
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

  function initializeConfirmations(root) {
    root.querySelectorAll('form[data-confirm]:not([data-ready])').forEach((form) => {
      form.dataset.ready = 'true';
      form.addEventListener('submit', (event) => {
        if (form.dataset.confirmed === 'true') return;
        event.preventDefault();
        showConfirm(form.dataset.confirm, () => {
          form.dataset.confirmed = 'true';
          form.requestSubmit(event.submitter);
        });
      });
    });
    root.querySelectorAll('button[data-confirm]:not([data-ready])').forEach((button) => {
      button.dataset.ready = 'true';
      button.addEventListener('click', (event) => {
        if (button.dataset.confirmed === 'true') return;
        event.preventDefault();
        showConfirm(button.dataset.confirm, () => {
          button.dataset.confirmed = 'true';
          button.form?.requestSubmit(button);
        });
      });
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

  function selection() {
    try {
      const value = JSON.parse(sessionStorage.getItem(selectionKey) || '[]');
      if (!Array.isArray(value)) return [];
      const seen = new Set();
      return value.flatMap((item) => {
        const number = String(item?.number || '').trim();
        const category = String(item?.category ?? '').trim();
        if (!number || !category || seen.has(number)) return [];
        seen.add(number);
        return [{ number, category, categoryName: String(item.categoryName || '') }];
      });
    } catch {
      return [];
    }
  }

  function storeSelection(value) {
    try { sessionStorage.setItem(selectionKey, JSON.stringify(value)); } catch { /* selection still works until navigation */ }
  }

  function selectionURL(value) {
    const params = new URLSearchParams();
    value.forEach((item) => params.append('order', item.number));
    return `/orders/selection?${params}`;
  }

  function selectionMode() {
    try { return sessionStorage.getItem(selectionModeKey) === 'true'; } catch { return false; }
  }

  function storeSelectionMode(value) {
    try { sessionStorage.setItem(selectionModeKey, String(value)); } catch { /* selection still works until navigation */ }
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
