import { useEffect, useMemo, useState } from 'react';
import { fetchCategories, fetchOrders } from '../../api/orders';
import { buildTableColumns, buildTableGroups } from './ordersTableModel';

export function useOrdersTable(catalog) {
  const [shift, setShift] = useState(0);
  const [orders, setOrders] = useState([]);
  const [categories, setCategories] = useState([]);
  const [activeCategoryID, setActiveCategoryID] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const columns = useMemo(() => buildTableColumns(shift), [shift]);

  useEffect(() => {
    fetchCategories()
      .then((rows) => setCategories(Array.isArray(rows) ? rows : []))
      .catch(() => setCategories([]));
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError('');
      try {
        const filters = { fulfillmentFrom: columns[0].key, fulfillmentTo: columns[columns.length - 1].key };
        const all = [];
        for (let page = 1; page <= 10; page += 1) {
          const result = await fetchOrders(page, 100, filters);
          all.push(...(result.items || []));
          if (all.length >= (result.total || 0) || !(result.items || []).length) break;
        }
        if (!cancelled) setOrders(all);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [columns]);

  const groups = useMemo(
    () => buildTableGroups(orders, columns, catalog, categories),
    [orders, columns, catalog, categories],
  );
  const activeGroup = useMemo(() => {
    if (!groups.length) return null;
    return groups.find((group) => (group.category?.id || 0) === activeCategoryID) || groups[0];
  }, [groups, activeCategoryID]);

  return {
    activeGroup,
    columns,
    error,
    groups,
    loading,
    orders,
    setActiveCategoryID,
    setShift,
    shift,
  };
}
