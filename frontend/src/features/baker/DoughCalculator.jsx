import { useEffect, useMemo, useState } from 'react';
import { Button } from '../../ui/Button';
import { EmptyState } from '../../ui/EmptyState';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { Icon } from '../../ui/Icon';
import { panelClass, PanelHeader } from '../../ui/Panel';
import { MonitorReports } from './MonitorReports';
import { calcDough, fetchCatalog, fetchCategories } from '../../api/orders';
import { categoryStyle } from '../../lib/categories';
import { plural } from '../../lib/format';

const qtyInputClass =
  'h-10 w-20 rounded-md border border-input bg-card px-2 text-center text-input tabular-nums shadow-xs transition-colors focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/20 sm:h-9';

// DoughCalculator — «черновик» пекаря: ввёл количества → посчитал тесто.
// Ничего не создаёт и не сохраняет — считает по тем же техкартам и кодам
// теста, что мониторинг заказов. Блюда берутся из каталога выбранного типа.
export function DoughCalculator() {
  const [categories, setCategories] = useState([]);
  const [catalog, setCatalog] = useState([]);
  const [activeCategoryID, setActiveCategoryID] = useState(0);
  const [query, setQuery] = useState('');
  const [quantities, setQuantities] = useState({});
  const [monitor, setMonitor] = useState(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    Promise.all([fetchCategories(), fetchCatalog()])
      .then(([cats, dishes]) => {
        const list = Array.isArray(cats) ? cats : [];
        setCategories(list);
        setCatalog(Array.isArray(dishes) ? dishes : []);
        if (list.length) setActiveCategoryID(list[0].id);
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  // Блюда активного типа (без типа — видны в любом), плюс поиск по имени.
  const dishes = useMemo(() => {
    const q = query.trim().toLowerCase();
    return catalog.filter((dish) => {
      if (dish.category_id != null && dish.category_id !== activeCategoryID) return false;
      if (q && !String(dish.name || '').toLowerCase().includes(q)) return false;
      return true;
    });
  }, [catalog, activeCategoryID, query]);

  const entered = useMemo(
    () =>
      catalog
        .filter((dish) => Number(quantities[dish.code]) > 0)
        .map((dish) => ({ code: dish.code, product_name: dish.name, quantity: Number(quantities[dish.code]) })),
    [catalog, quantities],
  );

  function setQuantity(code, raw) {
    setMonitor(null);
    setQuantities((current) => ({ ...current, [code]: raw.replace(/\D/g, '') }));
  }

  function switchCategory(id) {
    // Количества у типов независимы: смена типа не теряет ввод, но расчёт
    // идёт только по блюдам активного типа — сбрасываем устаревший результат.
    setActiveCategoryID(id);
    setMonitor(null);
    setError('');
  }

  function reset() {
    setQuantities({});
    setMonitor(null);
    setError('');
  }

  async function calculate() {
    // Считаем только позиции активного типа — коды теста у типов свои.
    const items = entered.filter((item) => {
      const dish = catalog.find((d) => d.code === item.code);
      return dish && (dish.category_id == null || dish.category_id === activeCategoryID);
    });
    if (!items.length) return;
    setError('');
    setLoading(true);
    try {
      setMonitor(await calcDough(activeCategoryID, items));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  const enteredCount = entered.length;

  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-5 lg:px-6">
      <div className="mx-auto max-w-3xl space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h1 className="m-0 text-page font-semibold">Калькулятор теста</h1>
            <p className="m-0 text-note text-muted-foreground">Введите количества — расчёт по техкартам, ничего не создаётся.</p>
          </div>
          <Button onClick={reset} disabled={loading || (!enteredCount && !monitor)}>
            <Icon name="close" size={15} />
            Сбросить
          </Button>
        </div>

        <ErrorBanner error={error} />

        <div className="flex flex-wrap items-center gap-1.5">
          {categories.map((category) => {
            const style = categoryStyle(category);
            const active = category.id === activeCategoryID;
            return (
              <button
                key={category.id}
                type="button"
                onClick={() => switchCategory(category.id)}
                className={`inline-flex min-h-9 items-center gap-1.5 rounded-full border px-3 py-1 text-body font-semibold transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30 ${
                  active ? style.chipActive : 'border-input bg-card text-muted-foreground hover:border-stone-400'
                }`}
              >
                <span className={`h-2 w-2 rounded-full ${style.dot}`} aria-hidden="true" />
                {category.name}
              </button>
            );
          })}
        </div>

        <section className={panelClass}>
          <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
            <PanelHeader title="Сколько печь" eyebrow="Черновик — не заявка" />
            <input
              className="h-10 w-full rounded-md border border-input bg-card px-3 text-input shadow-xs transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/20 sm:h-9 sm:w-56"
              type="search"
              placeholder="Найти блюдо…"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </div>
          {dishes.length ? (
            <div className="divide-y divide-border">
              {dishes.map((dish) => (
                <label className="flex items-center justify-between gap-2 py-1.5" key={dish.code}>
                  <span className="min-w-0 break-words text-body text-stone-800">{dish.name}</span>
                  <input
                    className={qtyInputClass}
                    type="text"
                    inputMode="numeric"
                    aria-label={`${dish.name}: количество`}
                    placeholder="0"
                    value={quantities[dish.code] ?? ''}
                    onChange={(event) => setQuantity(dish.code, event.target.value)}
                  />
                </label>
              ))}
            </div>
          ) : (
            <EmptyState compact>{query ? 'Ничего не найдено.' : 'В каталоге нет блюд этого типа.'}</EmptyState>
          )}
        </section>

        <section className={panelClass}>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <PanelHeader
              title="Расчёт теста"
              eyebrow={enteredCount ? `${enteredCount} ${plural(enteredCount, 'позиция', 'позиции', 'позиций')}` : 'Позиции не введены'}
            />
            <Button variant="primary" onClick={calculate} disabled={loading || !enteredCount}>
              <Icon name="calculator" size={15} />
              {loading ? 'Считаем…' : 'Рассчитать тесто'}
            </Button>
          </div>
          <MonitorReports monitor={monitor} loading={loading} canCalculate={enteredCount > 0} />
        </section>
      </div>
    </section>
  );
}
