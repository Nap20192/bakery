import { Button } from '../../components/Button';
import { Icon } from '../../components/Icon';

export function OrderPagination({ loading, page, onPageChange }) {
  return (
    <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
      <Button onClick={() => onPageChange(page.page - 1)} disabled={loading || page.page <= 1}>
        <Icon name="chevronLeft" size={16} />
        Назад
      </Button>
      <span className="whitespace-nowrap text-center text-xs text-stone-500">
        <span className="tabular-nums">{page.page} / {page.total_pages || 1}</span>
        {page.total > 0 && <span className="block text-[11px] text-stone-400">всего {page.total}</span>}
      </span>
      <Button onClick={() => onPageChange(page.page + 1)} disabled={loading || page.page >= (page.total_pages || 1)}>
        Далее
        <Icon name="chevronRight" size={16} />
      </Button>
    </div>
  );
}
