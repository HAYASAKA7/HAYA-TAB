let tabs = [];
let categories = [];
let currentCategoryId = "";
let isEditMode = false;
let currentSettings = {};

// Initialize
window.onload = async () => {
    try {
        await refreshData();
        await loadSettings();
        
        // Event Listeners
        window.runtime.EventsOn("tab-updated", (updatedTab) => {
            refreshData();
        });
        window.runtime.EventsOn("cover-error", (msg) => {
            showToast(msg, "error");
        });
        window.runtime.EventsOn("sync-complete", (msg) => {
            showToast(msg);
            refreshData();
        });

        // Global Click to close context menu
        document.addEventListener('click', () => {
            document.getElementById('context-menu').classList.add('hidden');
        });

        // Blank Space Context Menu
        document.getElementById('view-home').addEventListener('contextmenu', (e) => {
            e.preventDefault();
            if (e.target.closest('.tab-card')) return;
            
            showContextMenu(e.pageX, e.pageY, [
                { label: "New Category", action: () => openCategoryModal() },
                { label: "Upload TAB", action: () => addTab(true) },
                { label: "Link Local TAB", action: () => addTab(false) },
            ]);
        });

    } catch (err) {
        console.error(err);
    }
};

async function refreshData() {
    tabs = await window.go.main.App.GetTabs();
    categories = await window.go.main.App.GetCategories();
    renderGrid();
}

function goHome() {
    currentCategoryId = "";
    renderGrid();
}

// --- Navigation & Views ---

function switchView(viewName) {
    // Hide all views
    document.querySelectorAll('.view').forEach(el => el.classList.add('hidden'));
    document.getElementById('pdf-views-container').innerHTML = ''; // Clear pdfs if navigating away (or keep them hidden? For now clear to save memory/simple logic)
    // Actually, if we switch between Home and Settings, we might want to keep PDF tabs alive.
    // Let's refine: "view-home" and "view-settings" are static. PDF views are dynamic.
    // We will hide all children of main-content that are .view
    
    // Toggle Sidebar active state
    document.querySelectorAll('.sidebar-item').forEach(el => el.classList.remove('active'));
    
    if (viewName === 'home') {
        document.getElementById('view-home').classList.remove('hidden');
        document.getElementById('nav-home').classList.add('active');
        // Hide all PDF views
        document.querySelectorAll('.pdf-view').forEach(el => el.classList.add('hidden'));
    } else if (viewName === 'settings') {
        document.getElementById('view-settings').classList.remove('hidden');
        document.getElementById('nav-settings').classList.add('active');
         // Hide all PDF views
         document.querySelectorAll('.pdf-view').forEach(el => el.classList.add('hidden'));
    } else if (viewName.startsWith('pdf-')) {
        // Show specific PDF view
        const tabId = viewName.split('-')[1];
        const pdfView = document.getElementById(`pdf-view-${tabId}`);
        if (pdfView) {
            pdfView.classList.remove('hidden');
            // active sidebar item
            const navItem = document.getElementById(`nav-pdf-${tabId}`);
            if(navItem) navItem.classList.add('active');
        }
    }
}


// --- Grid Rendering (Home) ---

let draggedItem = null;

function renderGrid() {
    const grid = document.getElementById('tab-grid');
    grid.innerHTML = '';

    // 1. Back Button
    if (currentCategoryId) {
        const backCard = document.createElement('div');
        backCard.className = 'tab-card folder back-folder';
        
        const currentCat = categories.find(c => c.id === currentCategoryId);
        const parentId = currentCat ? currentCat.parentId : "";
        
        backCard.ondragover = (e) => handleDragOver(e, backCard);
        backCard.ondragleave = (e) => handleDragLeave(e, backCard);
        backCard.ondrop = (e) => handleDrop(e, parentId, backCard);

        backCard.onclick = () => {
            currentCategoryId = parentId;
            renderGrid();
        };
        backCard.innerHTML = `
            <div class="cover-wrapper" style="font-size: 2rem;">⤴️</div>
            <div class="info"><div class="title">.. (Back)</div></div>
        `;
        grid.appendChild(backCard);
    }

    // 2. Categories
    const subCats = categories.filter(c => c.parentId === currentCategoryId);
    for (const cat of subCats) {
        const card = document.createElement('div');
        card.className = 'tab-card folder';
        card.draggable = true;
        
        card.ondragstart = (e) => handleDragStart(e, { type: 'cat', id: cat.id });
        card.ondragover = (e) => handleDragOver(e, card);
        card.ondragleave = (e) => handleDragLeave(e, card);
        card.ondrop = (e) => handleDrop(e, cat.id, card);

        card.onclick = () => {
            currentCategoryId = cat.id;
            renderGrid();
        };
        card.oncontextmenu = (e) => {
            e.preventDefault();
            e.stopPropagation();
            showContextMenu(e.pageX, e.pageY, [
                { label: "Open", action: () => { currentCategoryId = cat.id; renderGrid(); } },
                { label: "Rename", action: () => openCategoryModal(cat) },
                { label: "Delete Category", action: () => deleteCategory(cat.id) }
            ]);
        };

        card.innerHTML = `
             <div class="cover-wrapper">📂</div>
             <div class="info"><div class="title">${cat.name}</div></div>
        `;
        grid.appendChild(card);
    }
    
    // 3. Tabs
    const currentTabs = tabs.filter(t => (t.categoryId || "") === currentCategoryId);
    for (const tab of currentTabs) {
        const card = document.createElement('div');
        card.className = 'tab-card';
        card.draggable = true;
        
        card.ondragstart = (e) => handleDragStart(e, { type: 'tab', id: tab.id });
        card.onclick = () => openTab(tab.id);
        
        card.oncontextmenu = (e) => {
            e.preventDefault();
            e.stopPropagation();
            const items = [
                { label: "Open (System)", action: () => window.go.main.App.OpenTab(tab.id) },
                { label: "Open (Inner)", action: () => openInternalTab(tab) },
                { label: "Edit Metadata", action: () => editTab(tab.id) },
                { label: "Move to Category...", action: () => promptMoveTab(tab.id) },
                { label: "Export TAB", action: () => exportTab(tab.id) },
                { type: "separator" },
                { label: tab.isManaged ? "Delete TAB" : "Unlink TAB", action: () => deleteTab(tab.id) }
            ];
            showContextMenu(e.pageX, e.pageY, items);
        };

        let coverHtml = `<div class="placeholder-cover">🎵</div>`;
        if (tab.coverPath) {
            coverHtml = `<div class="placeholder-cover" data-cover="${tab.coverPath}">🎵</div>`;
        }

        card.innerHTML = `
            <div class="edit-btn" onclick="event.stopPropagation(); editTab('${tab.id}')">✎</div>
            <div class="cover-wrapper">
                ${coverHtml}
            </div>
            <div class="info">
                <div class="title" title="${tab.title}">${tab.title}</div>
                <div class="artist" title="${tab.artist}">${tab.artist}</div>
                <div class="type-badge">${tab.type}</div>
            </div>
        `;
        grid.appendChild(card);
    }
    
    loadCovers();
}

// --- Drag & Drop (Unchanged logic) ---
function handleDragStart(e, item) { draggedItem = item; e.dataTransfer.effectAllowed = 'move'; e.stopPropagation(); }
function handleDragOver(e, element) { e.preventDefault(); if (!draggedItem) return; if (draggedItem.type === 'cat' && draggedItem.id === element.dataset.id) return; element.classList.add('drag-over'); e.dataTransfer.dropEffect = 'move'; }
function handleDragLeave(e, element) { element.classList.remove('drag-over'); }
async function handleDrop(e, targetCategoryId, element) {
    e.preventDefault(); element.classList.remove('drag-over');
    if (!draggedItem) return;
    if (draggedItem.type === 'cat' && draggedItem.id === targetCategoryId) return;
    try {
        if (draggedItem.type === 'tab') await window.go.main.App.MoveTab(draggedItem.id, targetCategoryId);
        else await window.go.main.App.MoveCategory(draggedItem.id, targetCategoryId);
        showToast("Moved successfully"); refreshData();
    } catch (err) { showToast("Move failed: " + err, "error"); }
    draggedItem = null;
}
async function loadCovers() {
    const placeholders = document.querySelectorAll('.placeholder-cover[data-cover]');
    for (const p of placeholders) {
        const path = p.dataset.cover;
        try {
            const b64 = await window.go.main.App.GetCover(path);
            if (b64) p.innerHTML = `<img src="data:image/jpeg;base64,${b64}" class="cover-img" loading="lazy">`;
        } catch (e) { }
    }
}

// --- Tab Logic & Internal Viewer ---

async function openTab(id) {
    const tab = tabs.find(t => t.id === id);
    if (!tab) return;

    // Check Settings
    if (currentSettings.openMethod === "inner" && tab.type === "pdf") {
        openInternalTab(tab);
    } else {
        // System Default
        try {
            await window.go.main.App.OpenTab(id);
        } catch (err) {
            console.error(err);
            showToast("Failed to open tab", "error");
        }
    }
}

function base64ToBlob(base64, type = "application/pdf") {
    const binStr = atob(base64);
    const len = binStr.length;
    const arr = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        arr[i] = binStr.charCodeAt(i);
    }
    return new Blob([arr], { type: type });
}

async function openInternalTab(tab) {
    // Check if already open
    if (document.getElementById(`pdf-view-${tab.id}`)) {
        switchView(`pdf-${tab.id}`);
        return;
    }

    try {
        const contentBase64 = await window.go.main.App.GetTabContent(tab.id);
        const blob = base64ToBlob(contentBase64, "application/pdf");
        const blobUrl = URL.createObjectURL(blob);
        
        // 1. Create Sidebar Item
        const sidebarList = document.getElementById('opened-tabs-list');
        const item = document.createElement('div');
        item.className = 'sidebar-item';
        item.id = `nav-pdf-${tab.id}`;
        item.onclick = () => switchView(`pdf-${tab.id}`);
        item.innerHTML = `
            <span class="icon">📄</span> <span style="flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">${tab.title}</span>
            <div class="close-tab" onclick="event.stopPropagation(); closeInternalTab('${tab.id}')">✕</div>
        `;
        sidebarList.appendChild(item);

        // 2. Create View Container
        const container = document.getElementById('pdf-views-container');
        const view = document.createElement('div');
        view.className = 'view pdf-view hidden';
        view.id = `pdf-view-${tab.id}`;
        
        view.innerHTML = `
            <div class="pdf-container">
                <iframe src="${blobUrl}" class="pdf-frame"></iframe>
            </div>
        `;
        container.appendChild(view);

        // Switch to it
        switchView(`pdf-${tab.id}`);

    } catch (e) {
        showToast("Failed to load PDF: " + e, "error");
    }
}

function closeInternalTab(id) {
    // Remove Sidebar Item
    const navItem = document.getElementById(`nav-pdf-${id}`);
    if(navItem) navItem.remove();

    // Remove View
    const view = document.getElementById(`pdf-view-${id}`);
    if(view) view.remove();

    // Switch back to home
    switchView('home');
}

// --- Settings Logic ---

async function loadSettings() {
    currentSettings = await window.go.main.App.GetSettings();
    if (!currentSettings) return;

    // Apply Theme
    if (currentSettings.theme === 'light') {
        document.body.setAttribute('data-theme', 'light');
    } else if (currentSettings.theme === 'dark') {
        document.body.removeAttribute('data-theme'); // default is dark
    } else {
        // System - simplistic check
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
            document.body.setAttribute('data-theme', 'light');
        } else {
            document.body.removeAttribute('data-theme');
        }
    }

    // Apply Background
    const layout = document.getElementById('app-layout');
    if (currentSettings.background && currentSettings.bgType) {
        let bgUrl = currentSettings.background;
        if (currentSettings.bgType === 'local') {
             // We need to fetch it or use asset server. 
             // Currently backend doesn't serve arbitrary local files via http easily without assetserver config.
             // We can use GetCover trick (base64) or Wails runtime.
             // For simplicity, let's try to load it as base64 if local
             if(!bgUrl.startsWith("http")) {
                 // It's a path. Let's use the GetCover method which reads file to base64 (it's named GetCover but works for files)
                 // A cleaner way would be a new method GetFileBase64, but GetCover does exactly that.
                 try {
                     const b64 = await window.go.main.App.GetCover(bgUrl);
                     bgUrl = `data:image/jpeg;base64,${b64}`; // Assuming jpg/png
                 } catch(e) { console.error("Failed to load bg", e); }
             }
        }
        layout.style.backgroundImage = `url('${bgUrl}')`;
    } else {
        layout.style.backgroundImage = 'none';
    }

    // Populate UI
    document.getElementById('set-theme').value = currentSettings.theme || "system";
    document.getElementById('set-bg-type').value = currentSettings.bgType || "";
    document.getElementById('set-bg-val').value = currentSettings.background || "";
    uiToggleBgInput();

    const openMethodInputs = document.querySelectorAll('input[name="openMethod"]');
    for (const input of openMethodInputs) {
        if (input.value === (currentSettings.openMethod || "system")) input.checked = true;
    }

    document.getElementById('set-sync-strategy').value = currentSettings.syncStrategy || "skip";
    
    renderSyncPaths();
}

function renderSyncPaths() {
    const list = document.getElementById('sync-path-list');
    list.innerHTML = '';
    const paths = currentSettings.syncPaths || [];
    paths.forEach((path, index) => {
        const li = document.createElement('li');
        li.innerHTML = `
            <span>${path}</span>
            <span class="delete-icon" onclick="removeSyncPath(${index})">🗑️</span>
        `;
        list.appendChild(li);
    });
}

async function saveSettings() {
    currentSettings.theme = document.getElementById('set-theme').value;
    currentSettings.bgType = document.getElementById('set-bg-type').value;
    currentSettings.background = document.getElementById('set-bg-val').value;
    
    const openMethod = document.querySelector('input[name="openMethod"]:checked');
    currentSettings.openMethod = openMethod ? openMethod.value : "system";
    
    currentSettings.syncStrategy = document.getElementById('set-sync-strategy').value;
    // syncPaths is already updated in memory

    try {
        await window.go.main.App.SaveSettings(currentSettings);
        showToast("Settings saved");
        loadSettings(); // Re-apply theme/bg
    } catch(e) {
        alert("Error saving settings: " + e);
    }
}

function uiToggleBgInput() {
    const type = document.getElementById('set-bg-type').value;
    const wrapper = document.getElementById('bg-input-wrapper');
    const browseBtn = document.getElementById('btn-browse-bg');
    
    if (!type) {
        wrapper.classList.add('hidden');
    } else {
        wrapper.classList.remove('hidden');
        if (type === 'local') {
            browseBtn.style.display = 'inline-block';
        } else {
            browseBtn.style.display = 'none';
        }
    }
}

async function browseBg() {
    const path = await window.go.main.App.SelectImage();
    if (path) {
        document.getElementById('set-bg-val').value = path;
    }
}


async function addSyncPath() {
    const path = await window.go.main.App.SelectFolder();
    if (path) {
        if (!currentSettings.syncPaths) currentSettings.syncPaths = [];
        if (!currentSettings.syncPaths.includes(path)) {
            currentSettings.syncPaths.push(path);
            renderSyncPaths();
        }
    }
}

function removeSyncPath(index) {
    if (currentSettings.syncPaths) {
        currentSettings.syncPaths.splice(index, 1);
        renderSyncPaths();
    }
}

async function triggerSync() {
    showToast("Sync started...");
    try {
        await window.go.main.App.TriggerSync();
        // Result handled by event 'sync-complete'
    } catch(e) {
        alert("Sync error: " + e);
    }
}

// --- Common Helper (Context Menu) ---
function showContextMenu(x, y, items) {
    const menu = document.getElementById('context-menu');
    const list = document.getElementById('context-menu-items');
    list.innerHTML = '';
    items.forEach(item => {
        const li = document.createElement('li');
        if (item.type === 'separator') li.className = 'separator';
        else {
            li.innerText = item.label;
            li.onclick = () => { item.action(); menu.classList.add('hidden'); };
        }
        list.appendChild(li);
    });
    const menuWidth = 150; const menuHeight = items.length * 35;
    if (x + menuWidth > window.innerWidth) x -= menuWidth;
    if (y + menuHeight > window.innerHeight) y -= menuHeight;
    menu.style.left = x + 'px'; menu.style.top = y + 'px';
    menu.classList.remove('hidden');
}

function showToast(message, type = "info") {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = 'toast';
    if (type === 'error') toast.classList.add('error');
    toast.innerText = message;
    container.appendChild(toast);
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(20px)';
        toast.style.transition = 'all 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// --- Standard Tab Actions (Wrappers) ---
async function addTab(isUpload) {
    const path = await window.go.main.App.SelectFile();
    if (path) {
        const tabData = await window.go.main.App.ProcessFile(path);
        openModal(tabData, isUpload ? 'upload' : 'link');
    }
}
function editTab(id) { const t = tabs.find(x => x.id === id); if(t) openModal(t, 'edit'); }
function openModal(d, m) {
    isEditMode = (m === 'edit');
    document.getElementById('edit-path').value = d.filePath;
    document.getElementById('edit-copy').value = (m === 'upload') ? "true" : "false";
    document.getElementById('edit-title').value = d.title;
    document.getElementById('edit-artist').value = d.artist;
    document.getElementById('edit-album').value = d.album;
    document.getElementById('edit-type').value = d.type;
    document.getElementById('edit-country').value = d.country || "US";
    document.getElementById('edit-lang').value = d.language || "en_us";
    document.getElementById('edit-form').dataset.id = d.id;
    document.querySelector('#modal-overlay .modal h2').innerText = isEditMode ? "Edit Tab Metadata" : "Add New Tab";
    document.getElementById('modal-overlay').classList.remove('hidden');
}
function closeModal() { document.getElementById('modal-overlay').classList.add('hidden'); document.getElementById('edit-form').reset(); }
async function saveTab() {
    const f = document.getElementById('edit-form');
    const existing = tabs.find(t => t.id === f.dataset.id);
    const t = {
        id: f.dataset.id,
        title: document.getElementById('edit-title').value,
        artist: document.getElementById('edit-artist').value,
        album: document.getElementById('edit-album').value,
        filePath: document.getElementById('edit-path').value,
        type: document.getElementById('edit-type').value,
        isManaged: existing ? existing.isManaged : false,
        coverPath: existing ? existing.coverPath : "",
        categoryId: existing ? existing.categoryId : currentCategoryId,
        country: document.getElementById('edit-country').value,
        language: document.getElementById('edit-lang').value
    };
    try {
        if (isEditMode) await window.go.main.App.UpdateTab(t);
        else await window.go.main.App.SaveTab(t, document.getElementById('edit-copy').value === "true");
        showToast("Saved."); closeModal(); refreshData();
    } catch (err) { alert(err); }
}
async function deleteTab(id) { if(confirm("Delete?")) { await window.go.main.App.DeleteTab(id); refreshData(); } }
async function exportTab(id) { const d = await window.go.main.App.SelectFolder(); if(d) { await window.go.main.App.ExportTab(id, d); showToast("Exported"); } }
function openCategoryModal(cat=null) {
    const m = document.getElementById('category-modal');
    m.classList.remove('hidden');
    if(cat) { document.getElementById('cat-id').value=cat.id; document.getElementById('cat-name').value=cat.name; }
    else { document.getElementById('cat-id').value=""; document.getElementById('cat-name').value=""; }
    document.getElementById('cat-name').focus();
}
function closeCategoryModal() { document.getElementById('category-modal').classList.add('hidden'); }
async function saveCategory() {
    const id = document.getElementById('cat-id').value;
    const name = document.getElementById('cat-name').value;
    if(!name) return;
    try {
        await window.go.main.App.AddCategory({ id: id, name: name, parentId: id ? categories.find(c=>c.id===id).parentId : currentCategoryId });
        closeCategoryModal(); refreshData();
    } catch(e) { alert(e); }
}
async function deleteCategory(id) { if(confirm("Delete?")) { await window.go.main.App.DeleteCategory(id); refreshData(); } }
function promptMoveTab(id) {
    document.getElementById('move-tab-id').value = id;
    const s = document.getElementById('move-select'); s.innerHTML='<option value="">(Root)</option>';
    categories.forEach(c => { const o=document.createElement('option'); o.value=c.id; o.innerText=c.name; s.appendChild(o); }); // simplified path
    document.getElementById('move-modal').classList.remove('hidden');
}
async function saveMove() {
    try { await window.go.main.App.MoveTab(document.getElementById('move-tab-id').value, document.getElementById('move-select').value); document.getElementById('move-modal').classList.add('hidden'); refreshData(); } catch(e) { alert(e); }
}
