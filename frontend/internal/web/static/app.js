(() => {
  'use strict';

  const selectionKey = 'bakery:selected-orders';
  const selectionModeKey = 'bakery:selection-mode';
  const selectionFilterKey = 'bakery:selection-filter';

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
    initializeMenu(root);
    initializeConfirmations(root);
    initializeDialogs(root);
    initializeSelection(root);
    initializeSelectionRemoval(root);
    initializeCatalog(root);
    initializeProduction(root);
  }

  function initializeSelectionRemoval(root) {
    const buttons = [...root.querySelectorAll('[data-selection-remove]')];
    buttons.filter((button) => button.dataset.ready !== 'true').forEach((button) => {
      button.dataset.ready = 'true';
      button.addEventListener('click', () => {
        const stored = selection();
        const current = stored.length ? stored : buttons.map((item) => ({ number: item.dataset.selectionRemove, category: '' }));
        const remaining = current.filter((item) => item.number !== button.dataset.selectionRemove);
        sessionStorage.setItem(selectionKey, JSON.stringify(remaining));
        const params = new URLSearchParams();
        remaining.forEach((item) => params.append('order', item.number));
        window.location.assign(remaining.length ? `/orders/selection?${params}` : '/orders');
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

  function initializeMenu(root) {
    root.querySelectorAll('[data-menu-toggle]:not([data-ready])').forEach((button) => {
      button.dataset.ready = 'true';
      button.addEventListener('click', () => {
        const nav = document.getElementById(button.getAttribute('aria-controls'));
        const open = !nav?.classList.contains('is-open');
        nav?.classList.toggle('is-open', open);
        button.setAttribute('aria-expanded', String(open));
      });
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
    try { return JSON.parse(sessionStorage.getItem(selectionKey) || '[]'); } catch { return []; }
  }

  function initializeSelection(root) {
    const toggle = root.querySelector('[data-selection-toggle]');
    const cards = [...root.querySelectorAll('[data-select-order]')];
    if (!toggle || !cards.length) return;
    if (toggle.dataset.ready === 'true') return;
    toggle.dataset.ready = 'true';
    let mode = sessionStorage.getItem(selectionModeKey) === 'true';
    let selected = selection();
    const filter = root.querySelector('[data-selection-toolbar]')?.dataset.selectionFilter || '';
    const previousFilter = sessionStorage.getItem(selectionFilterKey);
    if (previousFilter !== null && previousFilter !== filter) {
      selected = [];
      sessionStorage.removeItem(selectionKey);
    }
    sessionStorage.setItem(selectionFilterKey, filter);

    const render = () => {
      document.body.classList.toggle('selection-active', mode);
      toggle.setAttribute('aria-pressed', String(mode));
      toggle.textContent = mode ? 'Завершить выбор' : 'Выбор партии';
      cards.forEach((card) => {
        const picked = selected.some((item) => item.number === card.dataset.number);
        card.classList.toggle('is-selected', picked);
        card.setAttribute('aria-pressed', String(picked));
      });
      const open = root.querySelector('[data-selection-open]');
      const count = root.querySelector('[data-selection-count]');
      if (count) count.textContent = String(selected.length);
      if (open) {
        open.hidden = selected.length === 0;
        const params = new URLSearchParams();
        selected.forEach((item) => params.append('order', item.number));
        open.href = `/orders/selection?${params}`;
      }
    };

    toggle.addEventListener('click', () => {
      mode = !mode;
      sessionStorage.setItem(selectionModeKey, String(mode));
      render();
    });
    cards.forEach((card) => card.addEventListener('click', (event) => {
      if (event.target.closest('button, input, select, textarea')) return;
      if (!mode) return;
      event.preventDefault();
      const number = card.dataset.number;
      const category = card.dataset.category;
      if (card.dataset.cancelled === 'true') {
        showToast('Отменённый заказ нельзя добавить в партию.');
        return;
      }
      if (card.dataset.sheet) {
        showToast(`Заказ уже входит в отработку №${card.dataset.sheet}.`);
        return;
      }
      const found = selected.findIndex((item) => item.number === number);
      if (found >= 0) selected.splice(found, 1);
      else if (selected.length && selected[0].category !== category) showToast('В одну партию можно выбрать заявки только одного типа.');
      else selected.push({ number, category });
      sessionStorage.setItem(selectionKey, JSON.stringify(selected));
      render();
    }));
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
        editor.querySelectorAll('.category-choice').forEach((choice) => choice.classList.toggle('is-selected', Boolean(choice.querySelector('input:checked'))));
        updateSummary(editor);
      };
      radios.forEach((radio) => radio.addEventListener('change', updateRows));
      search?.addEventListener('input', updateRows);
      editor.querySelectorAll('.quantity-input').forEach((input) => input.addEventListener('input', () => updateSummary(editor)));
      editor.querySelectorAll('[data-comment-toggle]').forEach((button) => button.addEventListener('click', () => {
        const label = button.closest('[data-catalog-row]')?.querySelector('.item-comment');
        if (!label) return;
        label.hidden = !label.hidden;
        if (!label.hidden) label.querySelector('input')?.focus();
      }));
      updateRows();
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
