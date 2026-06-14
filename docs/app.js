/**
 * SPACE EXPLORATION APP LOGIC
 * Combines Three.js Universe, GSAP Scrolling, and Data Terminal Fetching
 */

// --- 1. DATA CONFIGURATION ---
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

// --- 2. THREE.JS UNIVERSE SETUP ---
const canvas = document.getElementById('universe-canvas');
const scene = new THREE.Scene();
scene.fog = new THREE.FogExp2(0x030305, 0.0015);

const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 2000);
camera.position.set(0, 0, 50); // Start slightly pulled back

const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true });
renderer.setSize(window.innerWidth, window.innerHeight);
renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

const ambientLight = new THREE.AmbientLight(0x222233, 1);
scene.add(ambientLight);
const pointLight = new THREE.PointLight(0x4488ff, 2, 150);
scene.add(pointLight);

const planets = [];

// Create Star Field (Particles)
const particlesCount = 10000;
const posArray = new Float32Array(particlesCount * 3);
for(let i=0; i < particlesCount * 3; i+=3) {
    posArray[i] = (Math.random() - 0.5) * 150;     // x spread
    posArray[i+1] = (Math.random() - 0.5) * 150;   // y spread
    posArray[i+2] = (Math.random() - 1) * 800;     // z depth down to -800
}
const particlesGeo = new THREE.BufferGeometry();
particlesGeo.setAttribute('position', new THREE.BufferAttribute(posArray, 3));
const particlesMat = new THREE.PointsMaterial({ size: 0.1, color: 0xffffff, transparent: true, opacity: 0.6 });
const particlesMesh = new THREE.Points(particlesGeo, particlesMat);
scene.add(particlesMesh);

function createPlanet(size, color, zPos, xPos, yPos, hasRings) {
    const geo = new THREE.SphereGeometry(size, 32, 32);
    const mat = new THREE.MeshStandardMaterial({
        color: color, wireframe: true, transparent: true, opacity: 0.3, emissive: color, emissiveIntensity: 0.5
    });
    const mesh = new THREE.Mesh(geo, mat);
    mesh.position.set(xPos, yPos, zPos);

    const coreGeo = new THREE.SphereGeometry(size * 0.95, 16, 16);
    const coreMat = new THREE.MeshBasicMaterial({ color: 0x030305 });
    mesh.add(new THREE.Mesh(coreGeo, coreMat));

    if (hasRings) {
        const ringGeo = new THREE.RingGeometry(size * 1.5, size * 2.2, 64);
        const ringMat = new THREE.MeshBasicMaterial({ color: color, side: THREE.DoubleSide, transparent: true, opacity: 0.2, wireframe: true });
        const ring = new THREE.Mesh(ringGeo, ringMat);
        ring.rotation.x = Math.PI / 2.5;
        mesh.add(ring);
    }

    scene.add(mesh);
    planets.push(mesh);
    return mesh;
}

// Map planets to the Sections
createPlanet(10, 0x44aaff, -100, 20, -5, true);  // Sector I
createPlanet(15, 0xffaa44, -250, -25, 10, false); // Sector II
createPlanet(8, 0xaa44ff, -400, 15, -10, true);  // Sector III
createPlanet(25, 0xff4444, -600, 0, 0, false);    // Sector IV

// Window Resize
window.addEventListener('resize', () => {
    camera.aspect = window.innerWidth / window.innerHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(window.innerWidth, window.innerHeight);
});

// Render Loop
const clock = new THREE.Clock();
function tick() {
    const elapsedTime = clock.getElapsedTime();
    planets.forEach((p, i) => {
        p.rotation.y += 0.002 * (i % 2 === 0 ? 1 : -1);
        p.rotation.x += 0.001;
    });
    particlesMesh.rotation.z = elapsedTime * 0.02;

    // Update HUD Coordinates
    document.getElementById('coordinates').innerText = `Z: ${camera.position.z.toFixed(2)}`;

    // Light follows camera
    pointLight.position.set(camera.position.x, camera.position.y, camera.position.z - 20);

    renderer.render(scene, camera);
    window.requestAnimationFrame(tick);
}
tick();


// --- 3. GSAP SCROLL ANIMATIONS ---
gsap.registerPlugin(ScrollTrigger);

// Animate Camera Z position based on scroll
gsap.to(camera.position, {
    z: -650, // Travel deep into the Z axis
    ease: "none",
    scrollTrigger: {
        trigger: "#scroll-container",
        start: "top top",
        end: "bottom bottom",
        scrub: 1
    }
});

// Animate Camera X/Y slightly for cinematic drift
gsap.to(camera.position, {
    x: 5,
    y: 2,
    ease: "none",
    scrollTrigger: {
        trigger: "#scroll-container",
        start: "top top",
        end: "bottom bottom",
        scrub: 2
    }
});


// --- 4. DATA TERMINAL LOGIC ---
const terminal = document.getElementById('data-terminal');
const closeBtn = document.getElementById('close-terminal');
const searchInput = document.getElementById('terminal-search');
const container = document.getElementById('data-container');
const loading = document.getElementById('loading');
const errorDiv = document.getElementById('error');

// Open Terminal on Button Click
document.querySelectorAll('.explore-btn').forEach(btn => {
    btn.addEventListener('click', async (e) => {
        const target = e.target.getAttribute('data-target');
        if (!target || !DATA_SOURCES[target]) return;

        terminal.classList.remove('hidden');
        await loadData(DATA_SOURCES[target]);
    });
});

closeBtn.addEventListener('click', () => {
    terminal.classList.add('hidden');
    container.innerHTML = '';
    searchInput.value = '';
});

searchInput.addEventListener('input', (e) => {
    renderData(e.target.value.toLowerCase());
});

async function loadData(source) {
    try {
        container.innerHTML = '';
        errorDiv.classList.add('hidden');
        loading.classList.remove('hidden');
        searchInput.disabled = true;
        searchInput.value = '';
        currentData = null;
        currentType = source.type;

        const response = await fetch(RAW_BASE_URL + source.path);
        if (!response.ok) throw new Error(`Fetch Failed: ${response.status}`);

        if (source.type === 'json') {
            currentData = await response.json();
        } else {
            const text = await response.text();
            currentData = text.split(/\r?\n/).filter(line => line.trim() !== '');
        }

        searchInput.disabled = false;
        renderData('');

    } catch (err) {
        errorDiv.textContent = `SYSTEM ERROR: ${err.message}`;
        errorDiv.classList.remove('hidden');
    } finally {
        loading.classList.add('hidden');
    }
}

function renderData(query) {
    if (!currentData) return;
    container.innerHTML = '';

    if (currentType === 'text') renderTextList(query);
    else if (currentType === 'json') renderJson(query);
}

function renderTextList(query) {
    let filtered = currentData;
    if (query) filtered = currentData.filter(item => item.toLowerCase().includes(query));

    if (filtered.length === 0) {
        container.innerHTML = '<p class="mission-log">NO MATCHING ENTITIES FOUND IN SECTOR.</p>';
        return;
    }

    const ul = document.createElement('ul');
    ul.className = 'text-list';

    const maxItems = 1500; // Keep DOM light for terminal UI
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
        li.style.color = "var(--accent-blue)";
        li.textContent = `[ DATA TRUNCATED. ${filtered.length - maxItems} MORE LOGS EXIST. USE QUERY. ]`;
        ul.appendChild(li);
    }
    container.appendChild(ul);
}

function renderJson(query) {
    let dataToRender = currentData;

    if (query) {
        if (Array.isArray(currentData)) {
            dataToRender = currentData.filter(item => JSON.stringify(item).toLowerCase().includes(query));
        } else if (typeof currentData === 'object' && currentData !== null) {
            dataToRender = {};
            for (const [key, value] of Object.entries(currentData)) {
                if (key.toLowerCase().includes(query) || JSON.stringify(value).toLowerCase().includes(query)) {
                    dataToRender[key] = value;
                }
            }
        }
    }

    if ((Array.isArray(dataToRender) && dataToRender.length === 0) ||
        (typeof dataToRender === 'object' && dataToRender !== null && Object.keys(dataToRender).length === 0)) {
        container.innerHTML = '<p class="mission-log">NO MATCHING DATA FRAGMENTS FOUND.</p>';
        return;
    }

    // Try Table format first
    if (Array.isArray(dataToRender) && dataToRender.length > 0 && typeof dataToRender[0] === 'object' && dataToRender[0] !== null) {
        const headers = new Set();
        dataToRender.forEach(item => {
            if(typeof item === 'object' && item !== null) Object.keys(item).forEach(k => headers.add(k));
        });

        if (headers.size > 0 && headers.size < 10) {
            const table = document.createElement('table');
            const thead = document.createElement('thead');
            const trHead = document.createElement('tr');

            headers.forEach(h => {
                const th = document.createElement('th');
                th.textContent = h.toUpperCase();
                trHead.appendChild(th);
            });
            thead.appendChild(trHead);
            table.appendChild(thead);

            const tbody = document.createElement('tbody');
            const maxItems = 500;
            const itemsToRender = dataToRender.slice(0, maxItems);

            itemsToRender.forEach(item => {
                const tr = document.createElement('tr');
                headers.forEach(h => {
                    const td = document.createElement('td');
                    let val = item[h];
                    td.textContent = typeof val === 'object' ? JSON.stringify(val) : (val !== undefined ? val : '-');
                    tr.appendChild(td);
                });
                tbody.appendChild(tr);
            });
            table.appendChild(tbody);
            container.appendChild(table);

            if (dataToRender.length > maxItems) {
                const p = document.createElement('p');
                p.className = 'mission-log';
                p.style.textAlign = 'center';
                p.textContent = `[ DISPLAYING 500 OF ${dataToRender.length} FRAGMENTS ]`;
                container.appendChild(p);
            }
            return;
        }
    }

    // Fallback JSON format
    const pre = document.createElement('pre');
    pre.className = 'json-view';
    pre.innerHTML = syntaxHighlight(JSON.stringify(dataToRender, null, 2));
    container.appendChild(pre);
}

function syntaxHighlight(json) {
    json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return json.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, function (match) {
        let cls = 'number';
        if (/^"/.test(match)) cls = /:$/.test(match) ? 'key' : 'string';
        else if (/true|false/.test(match)) cls = 'boolean';
        else if (/null/.test(match)) cls = 'null';
        return '<span class="' + cls + '">' + match + '</span>';
    });
}