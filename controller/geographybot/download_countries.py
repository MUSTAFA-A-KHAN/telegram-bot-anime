import urllib.request
import json

url = "https://raw.githubusercontent.com/mledoze/countries/master/countries.json"
req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
with urllib.request.urlopen(req) as response:
    data = json.loads(response.read().decode())

simplified = []
for country in data:
    name = country.get("name", {}).get("common", "")
    capital = country.get("capital", [""])[0] if country.get("capital") else ""
    region = country.get("region", "")
    flag = country.get("flag", "")

    if name and capital and flag:
        simplified.append({
            "name": name,
            "capital": capital,
            "region": region,
            "flag": flag
        })

with open("controller/geographybot/countries.json", "w", encoding="utf-8") as f:
    json.dump(simplified, f, ensure_ascii=False, indent=2)

print(f"Saved {len(simplified)} countries.")
