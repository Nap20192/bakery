export const MATRIX_PAGE_LIMIT = 200;
export const MATRIX_WINDOW_DAYS = 5;
export const MATRIX_WINDOW_START_OFFSET = -1;

export function dateValue(offset = 0) {
  const date = new Date();
  date.setDate(date.getDate() + offset);
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${date.getFullYear()}-${month}-${day}`;
}

export function matrixWindow(startOffset) {
  return {
    fulfillmentFrom: dateValue(startOffset),
    fulfillmentTo: dateValue(startOffset + MATRIX_WINDOW_DAYS - 1),
  };
}
