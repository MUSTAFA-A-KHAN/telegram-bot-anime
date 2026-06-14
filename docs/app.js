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
scene.fog = new THREE.FogExp2(0x050510, 0.001); // Thinner fog to see further

const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 3000);
camera.position.set(0, 0, 100); // Start pulled further back

const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: false }); // Solid bg for space
renderer.setClearColor(0x030305, 1);
renderer.setSize(window.innerWidth, window.innerHeight);
renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

const ambientLight = new THREE.AmbientLight(0xffffff, 0.3);
scene.add(ambientLight);
const pointLight = new THREE.PointLight(0x4488ff, 3, 300);
scene.add(pointLight);

// Make starfield larger and brighter
const particlesCount = 20000;
const posArray = new Float32Array(particlesCount * 3);
const colorsArray = new Float32Array(particlesCount * 3);

for(let i=0; i < particlesCount * 3; i+=3) {
    posArray[i] = (Math.random() - 0.5) * 500;     // Wider x spread
    posArray[i+1] = (Math.random() - 0.5) * 500;   // Wider y spread
    posArray[i+2] = (Math.random() - 1) * 2000;    // Deep Z depth

    // Give some stars a slight blue/orange tint
    const colorType = Math.random();
    if (colorType > 0.9) {
        colorsArray[i] = 0.4; colorsArray[i+1] = 0.6; colorsArray[i+2] = 1.0; // Blue
    } else if (colorType > 0.8) {
        colorsArray[i] = 1.0; colorsArray[i+1] = 0.8; colorsArray[i+2] = 0.5; // Orange
    } else {
        colorsArray[i] = 1.0; colorsArray[i+1] = 1.0; colorsArray[i+2] = 1.0; // White
    }
}
const particlesGeo = new THREE.BufferGeometry();
particlesGeo.setAttribute('position', new THREE.BufferAttribute(posArray, 3));
particlesGeo.setAttribute('color', new THREE.BufferAttribute(colorsArray, 3));

// Create a glowing texture for stars
const canvasTexture = document.createElement('canvas');
canvasTexture.width = 16; canvasTexture.height = 16;
const ctx = canvasTexture.getContext('2d');
const gradient = ctx.createRadialGradient(8, 8, 0, 8, 8, 8);
gradient.addColorStop(0, 'rgba(255,255,255,1)');
gradient.addColorStop(1, 'rgba(255,255,255,0)');
ctx.fillStyle = gradient;
ctx.fillRect(0,0,16,16);
const starTexture = new THREE.CanvasTexture(canvasTexture);

const particlesMat = new THREE.PointsMaterial({
    size: 1.5,
    vertexColors: true,
    transparent: true,
    opacity: 0.9,
    map: starTexture,
    blending: THREE.AdditiveBlending,
    depthWrite: false
});
const particlesMesh = new THREE.Points(particlesGeo, particlesMat);
scene.add(particlesMesh);

// Add massive glowing Nebulae/Galaxies
const nebulae = [];
for(let i=0; i<5; i++) {
    const geo = new THREE.PlaneGeometry(300, 300);
    const mat = new THREE.MeshBasicMaterial({
        map: starTexture, // Reusing radial gradient for a soft glow
        color: new THREE.Color().setHSL(Math.random() * 0.3 + 0.5, 1, 0.5), // Blue/Purple hues
        transparent: true,
        opacity: 0.15,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
        side: THREE.DoubleSide
    });
    const nebula = new THREE.Mesh(geo, mat);
    nebula.position.set((Math.random() - 0.5) * 400, (Math.random() - 0.5) * 400, -200 - (Math.random() * 1000));
    scene.add(nebula);
    nebulae.push(nebula);
}

// Giant cinematic planets
const planets = [];

function createPlanet(size, color, zPos, xPos, yPos, hasRings, hasMoons) {
    const geo = new THREE.SphereGeometry(size, 64, 64);

    // Solid core with emissive glow
    const coreMat = new THREE.MeshStandardMaterial({
        color: 0x111115,
        emissive: color,
        emissiveIntensity: 0.2,
        roughness: 0.8
    });
    const planet = new THREE.Mesh(geo, coreMat);
    planet.position.set(xPos, yPos, zPos);

    // Wireframe outer atmosphere layer
    const atmGeo = new THREE.SphereGeometry(size * 1.05, 32, 32);
    const atmMat = new THREE.MeshBasicMaterial({
        color: color, wireframe: true, transparent: true, opacity: 0.2
    });
    planet.add(new THREE.Mesh(atmGeo, atmMat));

    if (hasRings) {
        const ringGeo = new THREE.RingGeometry(size * 1.4, size * 2.8, 128);
        const ringMat = new THREE.MeshBasicMaterial({
            color: color,
            side: THREE.DoubleSide,
            transparent: true,
            opacity: 0.4,
            map: starTexture, // Soften the rings
            blending: THREE.AdditiveBlending
        });
        const ring = new THREE.Mesh(ringGeo, ringMat);
        ring.rotation.x = Math.PI / 2.2;
        ring.rotation.y = Math.PI / 8;
        planet.add(ring);

        // Add a secondary inner ring
        const ringGeo2 = new THREE.RingGeometry(size * 1.1, size * 1.3, 64);
        const ringMat2 = new THREE.MeshBasicMaterial({ color: 0xffffff, side: THREE.DoubleSide, transparent: true, opacity: 0.1 });
        const ring2 = new THREE.Mesh(ringGeo2, ringMat2);
        ring2.rotation.copy(ring.rotation);
        planet.add(ring2);
    }

    if (hasMoons) {
        for(let i=0; i<3; i++) {
            const moonGeo = new THREE.SphereGeometry(size * 0.1, 16, 16);
            const moonMat = new THREE.MeshStandardMaterial({ color: 0xdddddd, roughness: 0.9 });
            const moon = new THREE.Mesh(moonGeo, moonMat);

            // Create a pivot for the moon to orbit
            const pivot = new THREE.Group();
            pivot.rotation.x = Math.random() * Math.PI;
            pivot.rotation.y = Math.random() * Math.PI;
            pivot.add(moon);
            moon.position.set(size * 3 + Math.random() * size, 0, 0);

            planet.add(pivot);
            // Save pivot to planets array for rotation in render loop
            planets.push({ mesh: pivot, speed: 0.005 + Math.random() * 0.01 });
        }
    }

    scene.add(planet);
    planets.push({ mesh: planet, speed: 0.001 });
    return planet;
}

// Create Massive Planets directly in the camera's path
createPlanet(30, 0x44aaff, -150, 45, -15, true, true);   // Sector I: Geography (Blue with rings and moons)
createPlanet(45, 0xffaa44, -450, -60, 20, false, true);   // Sector II: Words (Giant orange gas giant)
createPlanet(25, 0xaa44ff, -750, 40, -10, true, false);  // Sector III: Scramy (Purple ringed)
createPlanet(80, 0xff2222, -1100, 0, 0, false, false);   // Sector IV: The Void (Massive Red giant)

// Add a Supermassive Black Hole accretion disk effect far away
const bhGeo = new THREE.RingGeometry(90, 150, 128);
const bhMat = new THREE.MeshBasicMaterial({
    color: 0xffaa00,
    side: THREE.DoubleSide,
    transparent: true,
    opacity: 0.8,
    blending: THREE.AdditiveBlending,
    map: starTexture
});
const blackHole = new THREE.Mesh(bhGeo, bhMat);
blackHole.position.set(0, 0, -1100);
blackHole.rotation.x = Math.PI / 2.5;
scene.add(blackHole);


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

    // Rotate Planets and Moons
    planets.forEach((p, i) => {
        p.mesh.rotation.y += p.speed;
        if(p.mesh.geometry && p.mesh.geometry.type === 'SphereGeometry') {
            p.mesh.rotation.x += p.speed * 0.5;
        }
    });

    blackHole.rotation.z -= 0.005;

    // Move particles slightly towards camera
    particlesMesh.position.z = (elapsedTime * 5) % 500;

    // Rotate Nebulae slowly to face camera
    nebulae.forEach((n, i) => {
        n.rotation.z += 0.0005 * (i % 2 === 0 ? 1 : -1);
        n.lookAt(camera.position);
    });

    // Update HUD Coordinates
    document.getElementById('coordinates').innerText = `Z: ${camera.position.z.toFixed(2)}`;

    // Light follows camera closely to illuminate planets as we pass
    pointLight.position.set(camera.position.x, camera.position.y + 10, camera.position.z - 50);

    renderer.render(scene, camera);
    window.requestAnimationFrame(tick);
}
tick();


// --- 3. GSAP SCROLL ANIMATIONS ---
gsap.registerPlugin(ScrollTrigger);

// Animate Camera Z position based on scroll deep into space
gsap.to(camera.position, {
    z: -1000,
    ease: "power1.inOut",
    scrollTrigger: {
        trigger: "#scroll-container",
        start: "top top",
        end: "bottom bottom",
        scrub: 1.5
    }
});

// Add camera shake/drift for cinematic effect
gsap.to(camera.position, {
    x: 10,
    y: 5,
    ease: "sine.inOut",
    scrollTrigger: {
        trigger: "#scroll-container",
        start: "top top",
        end: "bottom bottom",
        scrub: 3
    }
});


// --- 4. TELEGRAM WEB APP INIT ---
if (window.Telegram && window.Telegram.WebApp) {
    const tg = window.Telegram.WebApp;
    tg.ready();

    if (tg.initDataUnsafe && tg.initDataUnsafe.user) {
        const greetingElement = document.getElementById('user-greeting');
        if (greetingElement) {
            greetingElement.textContent = `WELCOME, ${tg.initDataUnsafe.user.first_name.toUpperCase()}!`;
            greetingElement.style.display = 'block';
        }
    }
}

// --- 5. DATA TERMINAL LOGIC ---
const terminal = document.getElementById('data-terminal');
const closeBtn = document.getElementById('close-terminal');
const searchInput = document.getElementById('terminal-search');
const container = document.getElementById('data-container');
const loading = document.getElementById('loading');
const errorDiv = document.getElementById('error');

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

    const maxItems = 1500;
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

    if (Array.isArray(dataToRender) && dataToRender.length > 0 && typeof dataToRender[0] === 'object' && dataToRender[0] !== null) {
        const headers = new Set();
        let hasImage = false;
        dataToRender.forEach(item => {
            if(typeof item === 'object' && item !== null) {
                Object.keys(item).forEach(k => {
                    headers.add(k);
                    if (k === 'image_url') hasImage = true;
                });
            }
        });

        if (hasImage || (headers.size > 0 && headers.size < 10)) {
            const grid = document.createElement('div');
            grid.className = 'card-grid';

            const maxItems = hasImage ? 100 : 500; // Limit cards to avoid performance issues
            const itemsToRender = dataToRender.slice(0, maxItems);

            itemsToRender.forEach(item => {
                const card = document.createElement('div');
                card.className = 'data-card';

                if (hasImage && item.image_url) {
                    const img = document.createElement('img');
                    img.src = item.image_url;
                    img.className = 'card-image';
                    img.alt = item.name || 'Data Image';
                    img.loading = 'lazy';
                    card.appendChild(img);
                }

                const content = document.createElement('div');
                content.className = 'card-content';

                headers.forEach(h => {
                    if (h === 'image_url') return; // Skip rendering URL as text
                    const val = item[h];
                    if (val !== undefined) {
                        const row = document.createElement('div');
                        row.className = 'card-row';

                        const labelSpan = document.createElement('span');
                        labelSpan.className = 'card-label';
                        labelSpan.textContent = h.toUpperCase() + ':';

                        const valueSpan = document.createElement('span');
                        valueSpan.className = 'card-value';
                        valueSpan.textContent = typeof val === 'object' ? JSON.stringify(val) : val;

                        row.appendChild(labelSpan);
                        // Add a small space between label and value
                        row.appendChild(document.createTextNode(' '));
                        row.appendChild(valueSpan);

                        content.appendChild(row);
                    }
                });

                card.appendChild(content);
                grid.appendChild(card);
            });

            container.appendChild(grid);

            if (dataToRender.length > maxItems) {
                const p = document.createElement('p');
                p.className = 'mission-log';
                p.style.textAlign = 'center';
                p.textContent = `[ DISPLAYING ${maxItems} OF ${dataToRender.length} FRAGMENTS ]`;
                container.appendChild(p);
            }
            return;
        }
    }

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