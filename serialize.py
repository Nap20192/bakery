import csv
import glob
import json
import re
import sys
from pathlib import Path


def find_csv(script_dir: Path) -> Path:
    """Ищет CSV-файл рядом со скриптом. Сначала аргумент, потом glob."""
    if len(sys.argv) > 1:
        p = Path(sys.argv[1])
        if p.exists():
            return p
        raise FileNotFoundError(f"Файл не найден: {p}")

    matches = list(script_dir.glob("*.csv"))
    if not matches:
        raise FileNotFoundError(
            f"CSV-файл не найден в {script_dir}\n"
            "Положите файл рядом со скриптом или передайте путь аргументом:\n"
            "  python convert.py /path/to/file.csv"
        )
    if len(matches) > 1:
        names = ", ".join(m.name for m in matches)
        raise FileNotFoundError(
            f"Найдено несколько CSV: {names}\n"
            "Укажите нужный явно: python convert.py /path/to/file.csv"
        )
    return matches[0]


def parse_float(s: str) -> float:
    """'0,540' -> 0.54  |  '  0,5400 ' -> 0.54  |  '-' -> 0.0"""
    s = s.strip().replace("\xa0", "").replace(" ", "")
    if not s or s == "-":
        return 0.0
    return float(s.replace(",", "."))


def is_ingredient_row(row: list[str]) -> bool:
    """Строка ингредиента: col[0] — целое число, col[1] — название продукта."""
    return bool(row[0].strip().isdigit() and row[1].strip())


def is_output_row(row: list[str]) -> bool:
    """Строка с выходом: col[17] содержит число, col[18] == 'кг'."""
    return (
        len(row) > 18
        and row[18].strip() in ("кг", "л")
        and row[17].strip() not in ("", "-")
    )


def build_products(path: str) -> list[dict]:
    products: dict[str, dict] = {}  # name -> product dict
    order: list[str] = []  # сохраняем порядок

    with open(path, newline="", encoding="utf-8-sig") as f:
        reader = csv.reader(f)
        rows = list(reader)

    for row in rows:
        # дополняем до 19 колонок на случай коротких строк
        while len(row) < 19:
            row.append("")

        if is_ingredient_row(row):
            product_name = row[1].strip()
            ing_name = row[2].strip()
            unit = row[6].strip()
            quantity = parse_float(row[7])
            gross = parse_float(row[9])
            net = parse_float(row[11])

            if product_name not in products:
                products[product_name] = {
                    "name": product_name,
                    "quantity": 0.0,
                    "unit": "кг",
                    "ingredients": [],
                }
                order.append(product_name)

            # последний ингредиент может «тянуть» ТРЕБОВАНИЯ К ОФОРМЛЕНИЮ —
            # обрезаем мусор из ing_name
            ing_name = re.sub(r"\s*\nТРЕБОВАНИЯ.*", "", ing_name, flags=re.S).strip()

            products[product_name]["ingredients"].append(
                {
                    "name": ing_name,
                    "quantity": quantity,
                    "unit": unit,
                    "gross": gross,
                    "net": net,
                }
            )

        elif is_output_row(row):
            # ищем, к какому продукту относится — берём последний открытый
            if order:
                last = order[-1]
                products[last]["quantity"] = parse_float(row[17])
                products[last]["unit"] = row[18].strip()

    return [products[n] for n in order]


def main():
    script_dir = Path(__file__).parent
    input_path = find_csv(script_dir)
    output_path = (
        Path(sys.argv[2]) if len(sys.argv) > 2 else script_dir / "products.json"
    )

    products = build_products(str(input_path))

    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(products, f, ensure_ascii=False, indent=2)

    print(f"✓ Прочитан: {input_path.name}")
    print(f"✓ Готово: {len(products)} продуктов → {output_path}")
    for p in products:
        print(
            f"  • {p['name']} ({p['quantity']} {p['unit']}) — {len(p['ingredients'])} ингр."
        )


if __name__ == "__main__":
    main()
