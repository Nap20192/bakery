import { formatQuantity } from './format';

export function orderQuantity(item) {
  if (!item.reserved_quantity) return formatQuantity(item.quantity);
  return `${formatQuantity(item.quantity)}+${formatQuantity(item.reserved_quantity)}`;
}

export function orderSource(order) {
  return order?.from_department?.name || order?.location || '-';
}

export function orderCreator(order) {
  if (order?.created_by_username) return `@${order.created_by_username}`;
  return orderSource(order);
}

function reportLines(reports) {
  return (reports || [])
    .filter(({ report }) => report?.ingredient?.quantity > 0)
    .flatMap(({ report }) => {
      const ing = report.ingredient;
      const lines = [`${ing.product_name}: ${formatQuantity(ing.quantity)} ${ing.unit}`.trim()];
      for (const component of report.components || []) {
        lines.push(`— ${component.product_name}: ${formatQuantity(component.quantity)} ${component.unit}`.trim());
      }
      return lines;
    });
}

// monitorToText renders a dough calculation as plain text for sharing.
export function monitorToText(monitor) {
  if (!monitor) return '';
  if (monitor.total_reports?.length) {
    const header = `Итого по заказам: ${monitor.orders?.length || 0}`;
    return [header, ...reportLines(monitor.total_reports)].join('\n');
  }
  if (monitor.reports?.length) {
    return reportLines(monitor.reports).join('\n');
  }
  return '';
}

// orderItemsToText renders order positions as plain text for sharing.
// Format matches the bot ("<name> <qty>[+<reserved>]") so copied text pastes
// back cleanly in both the web editor and the Telegram bot.
// Pass includeHeader=false to copy positions only, without the order title.
export function orderItemsToText(order, includeHeader = true) {
  const lines = (order?.items || []).map((item) => `${item.product_name} ${orderQuantity(item)}`);
  if (!includeHeader) return lines.join('\n');
  const header = order?.number ? `Заказ ${order.number}` : 'Заказ';
  return [header, ...lines].join('\n');
}
