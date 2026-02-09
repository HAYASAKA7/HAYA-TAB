let tabs = [];
let categories = [];
let currentCategoryId = "";
let isEditMode = false;
let currentSettings = {};
let openedTabs = []; // Track opened PDF tabs in order
let pinnedTabs = new Set(); // Track pinned tab IDs
let draggedSidebarItem = null; // For sidebar drag

// Batch Selection State
let isBatchSelectMode = false;
let selectedTabIds = new Set();

function toggleSidebar() {
    document.getElementById('sidebar').classList.toggle('collapsed');
}

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

// --- Custom Confirm Modal ---

function showConfirmModal(title, message, okText = "Confirm", isDanger = true) {
    return new Promise((resolve) => {
        const modal = document.getElementById('confirm-modal');
        const titleEl = document.getElementById('confirm-title');
        const messageEl = document.getElementById('confirm-message');
        const okBtn = document.getElementById('confirm-ok-btn');
        const cancelBtn = document.getElementById('confirm-cancel-btn');
        
        titleEl.textContent = title;
        messageEl.innerHTML = message;
        okBtn.textContent = okText;
        okBtn.className = isDanger ? 'btn danger' : 'btn primary';
        
        const cleanup = () => {
            modal.classList.add('hidden');
            okBtn.onclick = null;
            cancelBtn.onclick = null;
        };
        
        okBtn.onclick = () => {
            cleanup();
            resolve(true);
        };
        
        cancelBtn.onclick = () => {
            cleanup();
            resolve(false);
        };
        
        modal.classList.remove('hidden');
    });
}

async function refreshData() {
    tabs = await window.go.main.App.GetTabs() || [];
    categories = await window.go.main.App.GetCategories() || [];
    renderGrid();
    updateBatchActionBar();
}

function goHome() {
    currentCategoryId = "";
    renderGrid();
}

// --- Batch Selection Functions ---

function toggleBatchSelectMode() {
    isBatchSelectMode = !isBatchSelectMode;
    selectedTabIds.clear();
    renderGrid();
    updateBatchActionBar();
}

function exitBatchSelectMode() {
    isBatchSelectMode = false;
    selectedTabIds.clear();
    renderGrid();
    updateBatchActionBar();
}

function toggleTabSelection(tabId, event) {
    if (event) {
        event.stopPropagation();
    }
    if (selectedTabIds.has(tabId)) {
        selectedTabIds.delete(tabId);
    } else {
        selectedTabIds.add(tabId);
    }
    renderGrid();
    updateBatchActionBar();
}

function selectAllTabs() {
    const currentTabs = tabs.filter(t => (t.categoryId || "") === currentCategoryId);
    if (selectedTabIds.size === currentTabs.length) {
        // Deselect all
        selectedTabIds.clear();
    } else {
        // Select all
        currentTabs.forEach(t => selectedTabIds.add(t.id));
    }
    renderGrid();
    updateBatchActionBar();
}

function updateBatchActionBar() {
    const actionBar = document.getElementById('batch-action-bar');
    const countSpan = document.getElementById('batch-selected-count');
    
    if (isBatchSelectMode && selectedTabIds.size > 0) {
        actionBar.classList.remove('hidden');
        countSpan.textContent = selectedTabIds.size;
    } else {
        actionBar.classList.add('hidden');
    }
    
    // Update select mode button
    const selectBtn = document.getElementById('btn-select-mode');
    if (selectBtn) {
        selectBtn.classList.toggle('active', isBatchSelectMode);
        selectBtn.innerHTML = isBatchSelectMode 
            ? '<span class="icon-close"></span> Cancel'
            : '<span class="icon-checkbox"></span> Select';
    }
}

async function batchDeleteSelected() {
    if (selectedTabIds.size === 0) return;
    
    const selectedTabs = tabs.filter(t => selectedTabIds.has(t.id));
    const managedCount = selectedTabs.filter(t => t.isManaged).length;
    const linkedCount = selectedTabs.length - managedCount;
    
    let message = `You are about to remove <strong>${selectedTabIds.size}</strong> tab(s).`;
    if (managedCount > 0 && linkedCount > 0) {
        message += `<ul>
            <li><strong>${managedCount}</strong> uploaded tab(s) will be <span class="warning-text">deleted permanently</span></li>
            <li><strong>${linkedCount}</strong> linked tab(s) will be unlinked (files remain on disk)</li>
        </ul>`;
    } else if (managedCount > 0) {
        message += `<br><br>These <strong>${managedCount}</strong> uploaded tab(s) will be <span class="warning-text">deleted permanently</span>.`;
    } else {
        message += `<br><br>These <strong>${linkedCount}</strong> linked tab(s) will be unlinked (files remain on disk).`;
    }
    
    const confirmed = await showConfirmModal("Remove Tabs", message, "Remove", true);
    if (!confirmed) return;
    
    try {
        const ids = Array.from(selectedTabIds);
        const deleted = await window.go.main.App.BatchDeleteTabs(ids);
        showToast(`Removed ${deleted} tab(s)`);
        exitBatchSelectMode();
        refreshData();
    } catch (err) {
        showToast("Failed to delete tabs: " + err, "error");
    }
}

function openBatchMoveModal() {
    if (selectedTabIds.size === 0) return;
    
    const modal = document.getElementById('batch-move-modal');
    const select = document.getElementById('batch-move-select');
    
    // Build category tree options
    select.innerHTML = '<option value="">(Root)</option>';
    
    function buildCategoryPath(catId, path = []) {
        const cat = categories.find(c => c.id === catId);
        if (!cat) return path;
        path.unshift(cat.name);
        if (cat.parentId) {
            return buildCategoryPath(cat.parentId, path);
        }
        return path;
    }
    
    // Sort categories by their path
    const sortedCats = [...categories].sort((a, b) => {
        const pathA = buildCategoryPath(a.id).join('/');
        const pathB = buildCategoryPath(b.id).join('/');
        return pathA.localeCompare(pathB);
    });
    
    sortedCats.forEach(cat => {
        const path = buildCategoryPath(cat.id).join(' / ');
        const option = document.createElement('option');
        option.value = cat.id;
        option.textContent = path;
        select.appendChild(option);
    });
    
    modal.classList.remove('hidden');
}

function closeBatchMoveModal() {
    document.getElementById('batch-move-modal').classList.add('hidden');
}

async function saveBatchMove() {
    const categoryId = document.getElementById('batch-move-select').value;
    
    try {
        const ids = Array.from(selectedTabIds);
        const moved = await window.go.main.App.BatchMoveTabs(ids, categoryId);
        showToast(`Moved ${moved} tab(s)`);
        closeBatchMoveModal();
        exitBatchSelectMode();
        refreshData();
    } catch (err) {
        showToast("Failed to move tabs: " + err, "error");
    }
}

// --- Navigation & Views ---

function switchView(viewName) {
    // Hide all views
    document.querySelectorAll('.view').forEach(el => el.classList.add('hidden'));
    
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
            <div class="cover-wrapper"><span class="icon-back icon-xl"></span></div>
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
             <div class="cover-wrapper"><span class="icon-folder icon-xl"></span></div>
             <div class="info"><div class="title">${cat.name}</div></div>
        `;
        grid.appendChild(card);
    }
    
    // 3. Tabs
    const currentTabs = tabs.filter(t => (t.categoryId || "") === currentCategoryId);
    for (const tab of currentTabs) {
        const card = document.createElement('div');
        const isSelected = selectedTabIds.has(tab.id);
        card.className = 'tab-card' + (isBatchSelectMode && isSelected ? ' selected' : '');
        
        // In batch mode, allow dragging selected items to folders
        card.draggable = !isBatchSelectMode || (isBatchSelectMode && isSelected);
        
        if (isBatchSelectMode) {
            card.onclick = (e) => toggleTabSelection(tab.id, e);
            // Allow dragging selected items in batch mode
            if (isSelected) {
                card.ondragstart = (e) => {
                    e.dataTransfer.effectAllowed = 'move';
                    e.dataTransfer.setData('text/plain', 'batch');
                };
            }
        } else {
            card.ondragstart = (e) => handleDragStart(e, { type: 'tab', id: tab.id });
            card.onclick = () => openTab(tab.id);
        }
        
        card.oncontextmenu = (e) => {
            e.preventDefault();
            e.stopPropagation();
            if (isBatchSelectMode) return;
            const items = [
                { label: "Open with System", action: () => window.go.main.App.OpenTab(tab.id) },
                { label: "Open with Inner Viewer", action: () => openInternalTab(tab) },
                { label: "Edit Metadata", action: () => editTab(tab.id) },
                { label: "Move to Category...", action: () => promptMoveTab(tab.id) },
                { label: "Export TAB", action: () => exportTab(tab.id) },
                { type: "separator" },
                { label: tab.isManaged ? "Delete TAB" : "Unlink TAB", action: () => deleteTab(tab.id) }
            ];
            showContextMenu(e.pageX, e.pageY, items);
        };

        let coverHtml = `<div class="placeholder-cover"><span class="icon-music icon-xl"></span></div>`;
        if (tab.coverPath) {
            coverHtml = `<div class="placeholder-cover" data-cover="${tab.coverPath}"><span class="icon-music icon-xl"></span></div>`;
        }

        const checkboxHtml = isBatchSelectMode 
            ? `<div class="select-checkbox ${isSelected ? 'checked' : ''}" onclick="toggleTabSelection('${tab.id}', event)">
                <span class="icon-checkbox"></span>
               </div>`
            : '';

        const editBtnHtml = !isBatchSelectMode 
            ? `<div class="edit-btn" onclick="event.stopPropagation(); editTab('${tab.id}')"><span class="icon-edit"></span></div>`
            : '';

        card.innerHTML = `
            ${checkboxHtml}
            ${editBtnHtml}
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
function handleDragStart(e, item) { 
    draggedItem = item; 
    e.dataTransfer.effectAllowed = 'move'; 
    e.stopPropagation(); 
}
function handleDragOver(e, element) { 
    e.preventDefault(); 
    if (!draggedItem && selectedTabIds.size === 0) return; 
    if (draggedItem && draggedItem.type === 'cat' && draggedItem.id === element.dataset.id) return; 
    element.classList.add('drag-over'); 
    e.dataTransfer.dropEffect = 'move'; 
}
function handleDragLeave(e, element) { element.classList.remove('drag-over'); }
async function handleDrop(e, targetCategoryId, element) {
    e.preventDefault(); 
    element.classList.remove('drag-over');
    
    // Handle batch drag (if items are selected in batch mode and we're dragging to a category)
    if (isBatchSelectMode && selectedTabIds.size > 0) {
        try {
            const ids = Array.from(selectedTabIds);
            const moved = await window.go.main.App.BatchMoveTabs(ids, targetCategoryId);
            showToast(`Moved ${moved} tab(s)`);
            exitBatchSelectMode();
            refreshData();
        } catch (err) {
            showToast("Move failed: " + err, "error");
        }
        return;
    }
    
    if (!draggedItem) return;
    if (draggedItem.type === 'cat' && draggedItem.id === targetCategoryId) return;
    try {
        if (draggedItem.type === 'tab') await window.go.main.App.MoveTab(draggedItem.id, targetCategoryId);
        else await window.go.main.App.MoveCategory(draggedItem.id, targetCategoryId);
        showToast("Moved successfully"); refreshData();
    } catch (err) { showToast("Move failed: " + err, "error"); }
    draggedItem = null;
}

// --- Sidebar Tab Drag & Drop ---
function handleSidebarDragStart(e, tabId) {
    draggedSidebarItem = tabId;
    e.dataTransfer.effectAllowed = 'move';
    e.target.classList.add('dragging');
}

function handleSidebarDragEnd(e) {
    e.target.classList.remove('dragging');
    draggedSidebarItem = null;
    document.querySelectorAll('.sidebar-item.drag-over').forEach(el => el.classList.remove('drag-over'));
}

function handleSidebarDragOver(e, targetTabId) {
    e.preventDefault();
    if (!draggedSidebarItem || draggedSidebarItem === targetTabId) return;
    e.target.closest('.sidebar-item')?.classList.add('drag-over');
    e.dataTransfer.dropEffect = 'move';
}

function handleSidebarDragLeave(e) {
    e.target.closest('.sidebar-item')?.classList.remove('drag-over');
}

function handleSidebarDrop(e, targetTabId) {
    e.preventDefault();
    const dropTarget = e.target.closest('.sidebar-item');
    if (dropTarget) dropTarget.classList.remove('drag-over');
    
    if (!draggedSidebarItem || draggedSidebarItem === targetTabId) return;
    
    // Reorder openedTabs array
    const fromIndex = openedTabs.indexOf(draggedSidebarItem);
    const toIndex = openedTabs.indexOf(targetTabId);
    
    if (fromIndex !== -1 && toIndex !== -1) {
        openedTabs.splice(fromIndex, 1);
        openedTabs.splice(toIndex, 0, draggedSidebarItem);
        renderSidebarTabs();
    }
    
    draggedSidebarItem = null;
}

function showSidebarTabContextMenu(e, tabId) {
    e.preventDefault();
    e.stopPropagation();
    
    const isPinned = pinnedTabs.has(tabId);
    const items = [
        { label: isPinned ? "Unpin" : "Pin", action: () => togglePinTab(tabId) },
        { label: "Close", action: () => closeInternalTab(tabId) }
    ];
    
    showContextMenu(e.pageX, e.pageY, items);
}

function togglePinTab(tabId) {
    if (pinnedTabs.has(tabId)) {
        pinnedTabs.delete(tabId);
        showToast("Tab unpinned");
    } else {
        pinnedTabs.add(tabId);
        showToast("Tab pinned");
    }
    renderSidebarTabs();
}

function renderSidebarTabs() {
    const sidebarList = document.getElementById('opened-tabs-list');
    sidebarList.innerHTML = '';
    
    // Sort: pinned tabs first, then maintain order
    const sortedTabs = [...openedTabs].sort((a, b) => {
        const aPinned = pinnedTabs.has(a);
        const bPinned = pinnedTabs.has(b);
        if (aPinned && !bPinned) return -1;
        if (!aPinned && bPinned) return 1;
        return 0;
    });
    
    for (const tabId of sortedTabs) {
        const tab = tabs.find(t => t.id === tabId);
        if (!tab) continue;
        
        const isPinned = pinnedTabs.has(tabId);
        const item = document.createElement('div');
        item.className = 'sidebar-item' + (isPinned ? ' pinned' : '');
        item.id = `nav-pdf-${tabId}`;
        item.draggable = true;
        item.dataset.tabId = tabId;
        
        // Click to switch view
        item.onclick = () => switchView(`pdf-${tabId}`);
        
        // Context menu (right-click)
        item.oncontextmenu = (e) => showSidebarTabContextMenu(e, tabId);
        
        // Drag events
        item.ondragstart = (e) => handleSidebarDragStart(e, tabId);
        item.ondragend = handleSidebarDragEnd;
        item.ondragover = (e) => handleSidebarDragOver(e, tabId);
        item.ondragleave = handleSidebarDragLeave;
        item.ondrop = (e) => handleSidebarDrop(e, tabId);
        
        item.innerHTML = `
            <span class="icon">${isPinned ? '<span class="icon-pin"></span>' : '<span class="icon-document"></span>'}</span>
            <span class="sidebar-label" style="flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">${tab.title}</span>
            <div class="close-tab" onclick="event.stopPropagation(); closeInternalTab('${tabId}')"><span class="icon-close"></span></div>
        `;
        sidebarList.appendChild(item);
    }
    
    // Update active state
    const activeNav = document.querySelector('.sidebar-item.active[id^="nav-pdf-"]');
    if (activeNav) {
        const activeId = activeNav.id.replace('nav-pdf-', '');
        document.querySelectorAll('.sidebar-item[id^="nav-pdf-"]').forEach(el => el.classList.remove('active'));
        const newActive = document.getElementById(`nav-pdf-${activeId}`);
        if (newActive) newActive.classList.add('active');
    }
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
        
        // Convert to Blob URL for PDF.js
        const blob = base64ToBlob(contentBase64, "application/pdf");
        const blobUrl = URL.createObjectURL(blob);

        // 1. Add to openedTabs and render sidebar
        if (!openedTabs.includes(tab.id)) {
            openedTabs.push(tab.id);
        }
        renderSidebarTabs();

        // 2. Create View Container
        const container = document.getElementById('pdf-views-container');
        const view = document.createElement('div');
        view.className = 'view pdf-view hidden';
        view.id = `pdf-view-${tab.id}`;
        
        view.innerHTML = `
            <div class="pdf-container">
                <iframe src="pdfjs/web/viewer.html?file=${encodeURIComponent(blobUrl)}" class="pdf-frame"></iframe>
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
    // Remove from openedTabs array
    const index = openedTabs.indexOf(id);
    if (index !== -1) {
        openedTabs.splice(index, 1);
    }
    
    // Remove from pinned tabs if pinned
    pinnedTabs.delete(id);

    // Remove View
    const view = document.getElementById(`pdf-view-${id}`);
    if(view) {
        const iframe = view.querySelector('iframe');
        if (iframe && iframe.src.startsWith('blob:')) {
            URL.revokeObjectURL(iframe.src);
        }
        view.remove();
    }
    
    // Re-render sidebar tabs
    renderSidebarTabs();

    // Switch back to home
    switchView('home');
}

// --- Settings Logic ---

async function loadSettings() {
    currentSettings = await window.go.main.App.GetSettings();
    if (!currentSettings) return;

    // Ensure syncPaths is always an array
    if (!currentSettings.syncPaths) {
        currentSettings.syncPaths = [];
    }

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
            <span class="delete-icon" onclick="removeSyncPath(${index})"><span class="icon-trash"></span></span>
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
        // Save current settings (including sync paths) before syncing
        await window.go.main.App.SaveSettings(currentSettings);
        const msg = await window.go.main.App.TriggerSync();
        showToast(msg);
        refreshData();
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
    const paths = await window.go.main.App.SelectFiles();
    if (paths && paths.length > 0) {
        let added = 0;
        let skipped = 0;
        for (const path of paths) {
            try {
                const tabData = await window.go.main.App.ProcessFile(path);
                await window.go.main.App.SaveTab(tabData, isUpload);
                added++;
            } catch (err) {
                console.warn("Skipped duplicate or error:", err);
                skipped++;
            }
        }
        await refreshData();
        if (skipped > 0) {
            showToast(`Added ${added} tab(s), ${skipped} skipped (duplicates)`, skipped > 0 ? "warning" : undefined);
        } else if (added > 0) {
            showToast(`Added ${added} tab(s)`);
        }
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
    } catch (err) { showToast(err, "error"); }
}
async function deleteTab(id) {
    const tab = tabs.find(t => t.id === id);
    if (!tab) return;
    
    const title = tab.isManaged ? "Delete Tab" : "Unlink Tab";
    const message = tab.isManaged 
        ? `Are you sure you want to delete "<strong>${tab.title}</strong>"?<br><br><span class="warning-text">This will permanently delete the file.</span>`
        : `Are you sure you want to unlink "<strong>${tab.title}</strong>"?<br><br>The file will remain on disk.`;
    const btnText = tab.isManaged ? "Delete" : "Unlink";
    
    const confirmed = await showConfirmModal(title, message, btnText, true);
    if (confirmed) {
        await window.go.main.App.DeleteTab(id);
        refreshData();
    }
}
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
    } catch(e) { showToast(e, "error"); }
}
async function deleteCategory(id) {
    const cat = categories.find(c => c.id === id);
    if (!cat) return;
    
    const message = `Are you sure you want to delete the category "<strong>${cat.name}</strong>"?<br><br>Tabs in this category will be moved to root.`;
    
    const confirmed = await showConfirmModal("Delete Category", message, "Delete", true);
    if (confirmed) {
        await window.go.main.App.DeleteCategory(id);
        refreshData();
    }
}
function promptMoveTab(id) {
    document.getElementById('move-tab-id').value = id;
    const s = document.getElementById('move-select'); s.innerHTML='<option value="">(Root)</option>';
    categories.forEach(c => { const o=document.createElement('option'); o.value=c.id; o.innerText=c.name; s.appendChild(o); }); // simplified path
    document.getElementById('move-modal').classList.remove('hidden');
}
async function saveMove() {
    try { await window.go.main.App.MoveTab(document.getElementById('move-tab-id').value, document.getElementById('move-select').value); document.getElementById('move-modal').classList.add('hidden'); refreshData(); } catch(e) { showToast(e, "error"); }
}
