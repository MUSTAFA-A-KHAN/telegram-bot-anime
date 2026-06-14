const REPO_OWNER = 'MUSTAFA-A-KHAN';
const REPO_NAME = 'telegram-bot-anime';
const BRANCH = 'main';
const RAW_BASE_URL = `https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/${BRANCH}/`;

const DATA_SOURCES = {
    'geography_countries': { path: 'controller/geographybot/countries.json', type: 'json' },
    'geography_landmarks': { path: 'controller/geographybot/landmarks.json', type: 'json' },
    'words_txt': { path: 'controller/translator/words.txt', type: 'text' },
    'allowed_words_txt': { path: 'controller/translator/allowed_words.txt', type: 'text' },
    'scramy_words_txt': { path: 'controller/translator/scramy_words.txt', type: 'text' },
    'scramy_allowed_words_txt': { path: 'controller/translator/scramy_allowed_words.txt', type: 'text' },
    'translator_banned_users': { path: 'translator_banned_users.json', type: 'json' }
};

let currentData = null;
let currentType = null;

const selector = document.getElementById('data-selector');
const searchInput = document.getElementById('search-input');
const container = document.getElementById('data-container');
const loading = document.getElementById('loading');
const errorDiv = document.getElementById('error');

selector.addEventListener('change', async (e) => {
    const key = e.target.value;
    if (!key || !DATA_SOURCES[key]) return;

    const source = DATA_SOURCES[key];
    await loadData(source);
});

searchInput.addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase();
    renderData(query);
});

async function loadData(source) {
    try {
        // Reset UI
        container.innerHTML = '';
        errorDiv.classList.add('hidden');
        loading.classList.remove('hidden');
        searchInput.disabled = true;
        searchInput.value = '';
        currentData = null;
        currentType = source.type;

        const response = await fetch(RAW_BASE_URL + source.path);

        if (!response.ok) {
            throw new Error(`Failed to fetch: ${response.status} ${response.statusText}`);
        }

        if (source.type === 'json') {
            currentData = await response.json();
        } else {
            const text = await response.text();
            // Split by newline and remove empty lines
            currentData = text.split(/\r?\n/).filter(line => line.trim() !== '');
        }

        searchInput.disabled = false;
        renderData('');

    } catch (err) {
        errorDiv.textContent = `Error loading data: ${err.message}`;
        errorDiv.classList.remove('hidden');
    } finally {
        loading.classList.add('hidden');
    }
}

function renderData(query) {
    if (!currentData) return;

    container.innerHTML = '';

    if (currentType === 'text') {
        renderTextList(query);
    } else if (currentType === 'json') {
        renderJson(query);
    }
}

function renderTextList(query) {
    let filtered = currentData;
    if (query) {
        filtered = currentData.filter(item => item.toLowerCase().includes(query));
    }

    if (filtered.length === 0) {
        container.innerHTML = '<p class="placeholder-text">No matching items found.</p>';
        return;
    }

    const ul = document.createElement('ul');
    ul.className = 'text-list';

    // For performance on huge lists, we could virtualize,
    // but for simple text lists up to ~10k items, DOM appending usually works ok.
    // Let's cap at 5000 for safety to avoid browser freeze on huge datasets.
    const maxItems = 5000;
    const itemsToRender = filtered.slice(0, maxItems);

    itemsToRender.forEach(item => {
        const li = document.createElement('li');
        li.textContent = item;
        ul.appendChild(li);
    });

    if (filtered.length > maxItems) {
        const li = document.createElement('li');
        li.style.gridColumn = "1 / -1";
        li.style.textAlign = "center";
        li.style.color = "#6c757d";
        li.textContent = `... and ${filtered.length - maxItems} more items (use search to filter)`;
        ul.appendChild(li);
    }

    container.appendChild(ul);
}

function renderJson(query) {
    let dataToRender = currentData;

    if (query) {
        if (Array.isArray(currentData)) {
            // Filter array of objects
            dataToRender = currentData.filter(item => {
                return JSON.stringify(item).toLowerCase().includes(query);
            });
        } else if (typeof currentData === 'object' && currentData !== null) {
            // Filter object keys/values
            dataToRender = {};
            for (const [key, value] of Object.entries(currentData)) {
                if (key.toLowerCase().includes(query) || JSON.stringify(value).toLowerCase().includes(query)) {
                    dataToRender[key] = value;
                }
            }
        }
    }

    // Check if empty
    if (Array.isArray(dataToRender) && dataToRender.length === 0) {
        container.innerHTML = '<p class="placeholder-text">No matching items found.</p>';
        return;
    }
    if (typeof dataToRender === 'object' && dataToRender !== null && Object.keys(dataToRender).length === 0) {
        container.innerHTML = '<p class="placeholder-text">No matching items found.</p>';
        return;
    }

    // If it's a flat array of similar objects, render as a table
    if (Array.isArray(dataToRender) && dataToRender.length > 0 && typeof dataToRender[0] === 'object' && dataToRender[0] !== null) {
        // Collect all unique keys for headers
        const headers = new Set();
        dataToRender.forEach(item => {
            if(typeof item === 'object' && item !== null) {
                Object.keys(item).forEach(k => headers.add(k));
            }
        });

        if (headers.size > 0 && headers.size < 10) { // arbitrary limit to not make tables too wide
            const table = document.createElement('table');
            const thead = document.createElement('thead');
            const trHead = document.createElement('tr');

            headers.forEach(h => {
                const th = document.createElement('th');
                th.textContent = h;
                trHead.appendChild(th);
            });
            thead.appendChild(trHead);
            table.appendChild(thead);

            const tbody = document.createElement('tbody');

            const maxItems = 1000;
            const itemsToRender = dataToRender.slice(0, maxItems);

            itemsToRender.forEach(item => {
                const tr = document.createElement('tr');
                headers.forEach(h => {
                    const td = document.createElement('td');
                    let val = item[h];
                    if (typeof val === 'object') {
                        td.textContent = JSON.stringify(val);
                    } else {
                        td.textContent = val !== undefined ? val : '';
                    }
                    tr.appendChild(td);
                });
                tbody.appendChild(tr);
            });
            table.appendChild(tbody);
            container.appendChild(table);

            if (dataToRender.length > maxItems) {
                const p = document.createElement('p');
                p.style.textAlign = 'center';
                p.style.padding = '1rem';
                p.textContent = `Showing 1000 of ${dataToRender.length} items. Use search to find more.`;
                container.appendChild(p);
            }
            return;
        }
    }

    // Fallback: render as syntax-highlighted JSON string
    const pre = document.createElement('pre');
    pre.className = 'json-view';
    const jsonString = JSON.stringify(dataToRender, null, 2);
    pre.innerHTML = syntaxHighlight(jsonString);
    container.appendChild(pre);
}

function syntaxHighlight(json) {
    json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return json.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, function (match) {
        var cls = 'number';
        if (/^"/.test(match)) {
            if (/:$/.test(match)) {
                cls = 'key';
            } else {
                cls = 'string';
            }
        } else if (/true|false/.test(match)) {
            cls = 'boolean';
        } else if (/null/.test(match)) {
            cls = 'null';
        }
        return '<span class="' + cls + '">' + match + '</span>';
    });
}
