import csv
import uuid

files = [
    "ФОРМУЛЯР_ПЕКАРИ_авг2024 - калькуляции тесто.csv",
    "ФОРМУЛЯР_ПЕКАРИ_авг2024 - выпечка.csv"
]

parsed_items = []

# Парсинг файлов
for file in files:
    with open(file, 'r', encoding='utf-8') as f:
        reader = csv.reader(f)
        for row in reader:
            if len(row) < 8: continue
            if not row[0].strip().isdigit(): continue # Берем только строки рецептуры (1, 2, 3...)

            parent = row[1].strip()
            ingredient = row[2].strip()
            unit = row[6].strip()
            # Убираем пробелы и меняем запятую на точку для Float
            amount_str = row[7].strip().replace(' ', '').replace(',', '.')

            try:
                amount = float(amount_str)
            except ValueError:
                continue

            if parent and ingredient and amount > 0:
                parsed_items.append({
                    'parent': parent,
                    'ingredient': ingredient,
                    'unit': unit,
                    'amount': amount
                })

# Словари для хранения UUID
products = {}
ingredients = {}
tech_cards = {}
tc_items_sql = []

def get_product(name):
    if name not in products: products[name] = str(uuid.uuid4())
    return products[name]

def get_ingredient(name):
    if name not in ingredients: ingredients[name] = str(uuid.uuid4())
    return ingredients[name]

def get_tech_card(parent_name):
    if parent_name not in tech_cards: tech_cards[parent_name] = str(uuid.uuid4())
    return tech_cards[parent_name]

def escape_sql(text):
    return text.replace("'", "''") # Экранирование кавычек для SQL

# Распределение связей
for item in parsed_items:
    parent_id = get_product(item['parent'])
    tc_id = get_tech_card(item['parent'])

    # Логика: если в названии ингредиента есть "п/ф" (полуфабрикат) — это саб-продукт
    is_sub_product = "п/ф" in item['ingredient'].lower()
    item_id = str(uuid.uuid4())

    if is_sub_product:
        sub_prod_id = get_product(item['ingredient'])
        tc_items_sql.append(f"('{item_id}', '{tc_id}', NULL, '{sub_prod_id}', {item['amount']}, {item['amount']}, '{item['unit']}')")
    else:
        ing_id = get_ingredient(item['ingredient'])
        tc_items_sql.append(f"('{item_id}', '{tc_id}', '{ing_id}', NULL, {item['amount']}, {item['amount']}, '{item['unit']}')")

# Генерация SQL-файла
with open("seed.sql", "w", encoding="utf-8") as f:
    f.write("BEGIN;\n\n")

    f.write("-- ПРОДУКТЫ И ПОЛУФАБРИКАТЫ\n")
    f.write("INSERT INTO products (id, name, unit) VALUES\n")
    prod_vals = [f"('{uid}', '{escape_sql(name)}', '{'кг' if 'п/ф' in name.lower() else 'шт'}')" for name, uid in products.items()]
    f.write(",\n".join(prod_vals) + "\nON CONFLICT (name) DO NOTHING;\n\n")

    f.write("-- ИНГРЕДИЕНТЫ\n")
    f.write("INSERT INTO ingredients (id, name, unit) VALUES\n")
    ing_vals = [f"('{uid}', '{escape_sql(name)}', 'кг')" for name, uid in ingredients.items()]
    f.write(",\n".join(ing_vals) + "\nON CONFLICT (name) DO NOTHING;\n\n")

    f.write("-- ТЕХНОЛОГИЧЕСКИЕ КАРТЫ\n")
    f.write("INSERT INTO tech_cards (id, product_id, yield_qty, yield_unit) VALUES\n")
    tc_vals = [f"('{uid}', '{products[name]}', 1.0000, '{'кг' if 'п/ф' in name.lower() else 'шт'}')" for name, uid in tech_cards.items()]
    f.write(",\n".join(tc_vals) + ";\n\n")

    f.write("-- СОСТАВ ТЕХ. КАРТ\n")
    f.write("INSERT INTO tech_card_items (id, tech_card_id, ingredient_id, sub_product_id, gross_qty, net_qty, unit) VALUES\n")
    f.write(",\n".join(tc_items_sql) + ";\n\n")

    f.write("COMMIT;\n")

print(f"Готово! SQL файл seed.sql сгенерирован. Обработано строк: {len(parsed_items)}")
