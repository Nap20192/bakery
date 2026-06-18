import { Button } from '../../components/Button';

export function OrderPagination({ loading, page, onPageChange }) {
  return (
    <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
      <Button onClick={() => onPageChange(page.page - 1)} disabled={loading || page.page <= 1}>
        Назад
      </Button>
      <span className="whitespace-nowrap text-xs text-stone-500">
        {page.page} / {page.total_pages || 1}
      </span>
      <Button onClick={() => onPageChange(page.page + 1)} disabled={loading || page.page >= (page.total_pages || 1)}>
        Далее
      </Button>
    </div>
  );
}
