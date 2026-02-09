let tabs = [];
let categories = [];
let currentCategoryId = "";
let isEditMode = false;

// Initialize
window.onload = async () => {
    try {
        await refreshData();
        
        // Event Listeners
        window.runtime.EventsOn("tab-updated", (updatedTab) => {
            console.log("Tab updated:", updatedTab);
            refreshData();
        });
        window.runtime.EventsOn("cover-error", (msg) => {
            showToast(msg, "error");
        });

        // Global Click to close context menu
        document.addEventListener('click', () => {
            document.getElementById('context-menu').classList.add('hidden');
        });

        // Blank Space Context Menu
        document.getElementById('app').addEventListener('contextmenu', (e) => {
            e.preventDefault();
            // If we clicked on a card, don't show blank menu (event propagation should be stopped by card, but just in case)
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

let draggedItem = null;

function renderGrid() {
    const grid = document.getElementById('tab-grid');
    grid.innerHTML = '';

    // 1. Back Button (if not root)
    if (currentCategoryId) {
        const backCard = document.createElement('div');
        backCard.className = 'tab-card folder back-folder';
        
        // Drag Target (Move to parent)
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

    // 2. Categories (Sub-folders)
    const subCats = categories.filter(c => c.parentId === currentCategoryId);
    for (const cat of subCats) {
        const card = document.createElement('div');
        card.className = 'tab-card folder';
        card.draggable = true;
        
        // Drag Events
        card.ondragstart = (e) => handleDragStart(e, { type: 'cat', id: cat.id });
        card.ondragover = (e) => handleDragOver(e, card);
        card.ondragleave = (e) => handleDragLeave(e, card);
        card.ondrop = (e) => handleDrop(e, cat.id, card);

        card.onclick = () => {
            currentCategoryId = cat.id;
            renderGrid();
        };
        // Right click on category
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
        
        // Drag Events
        card.ondragstart = (e) => handleDragStart(e, { type: 'tab', id: tab.id });

        card.onclick = () => openTab(tab.id);
        
        // Right click on Tab
        card.oncontextmenu = (e) => {
            e.preventDefault();
            e.stopPropagation();
            const items = [
                { label: "Open", action: () => openTab(tab.id) },
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
            // Check if we need to load it (it's base64 encoded by backend method)
            // Ideally we'd lazy load this.
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
    
    // Lazy load covers
    loadCovers();
}

// --- Drag and Drop ---

function handleDragStart(e, item) {
    draggedItem = item;
    e.dataTransfer.effectAllowed = 'move';
    e.stopPropagation();
}

function handleDragOver(e, element) {
    e.preventDefault();
    if (!draggedItem) return;
    // Don't drag category into itself
    if (draggedItem.type === 'cat' && draggedItem.id === element.dataset.id) return;
    
    element.classList.add('drag-over');
    e.dataTransfer.dropEffect = 'move';
}

function handleDragLeave(e, element) {
    element.classList.remove('drag-over');
}

async function handleDrop(e, targetCategoryId, element) {
    e.preventDefault();
    element.classList.remove('drag-over');
    
    if (!draggedItem) return;

    // Prevent circular logic (simple check)
    if (draggedItem.type === 'cat' && draggedItem.id === targetCategoryId) {
         return;
    }

    try {
        if (draggedItem.type === 'tab') {
            await window.go.main.App.MoveTab(draggedItem.id, targetCategoryId);
        } else {
            await window.go.main.App.MoveCategory(draggedItem.id, targetCategoryId);
        }
        showToast("Moved successfully");
        refreshData();
    } catch (err) {
        showToast("Move failed: " + err, "error");
    }
    draggedItem = null;
}

async function loadCovers() {
    const placeholders = document.querySelectorAll('.placeholder-cover[data-cover]');
    for (const p of placeholders) {
        const path = p.dataset.cover;
        try {
            const b64 = await window.go.main.App.GetCover(path);
            if (b64) {
                p.innerHTML = `<img src="data:image/jpeg;base64,${b64}" class="cover-img" loading="lazy">`;
            }
        } catch (e) {
            console.warn("Failed to load cover", e);
        }
    }
}

// --- Context Menu ---
function showContextMenu(x, y, items) {
    const menu = document.getElementById('context-menu');
    const list = document.getElementById('context-menu-items');
    list.innerHTML = '';
    
    items.forEach(item => {
        const li = document.createElement('li');
        if (item.type === 'separator') {
            li.className = 'separator';
        } else {
            li.innerText = item.label;
            li.onclick = () => {
                item.action();
                menu.classList.add('hidden');
            };
        }
        list.appendChild(li);
    });

    // Boundary check
    const menuWidth = 150; 
    const menuHeight = items.length * 35; 
    
    if (x + menuWidth > window.innerWidth) x -= menuWidth;
    if (y + menuHeight > window.innerHeight) y -= menuHeight;

    menu.style.left = x + 'px';
    menu.style.top = y + 'px';
    menu.classList.remove('hidden');
}

// --- Actions ---

async function addTab(isUpload) {
    try {
        const path = await window.go.main.App.SelectFile();
        if (path) {
            const tabData = await window.go.main.App.ProcessFile(path);
            openModal(tabData, isUpload ? 'upload' : 'link');
        }
    } catch (err) {
        console.error(err);
    }
}

function editTab(id) {
    const tab = tabs.find(t => t.id === id);
    if (tab) {
        openModal(tab, 'edit');
    }
}

function openModal(tabData, mode) {
    isEditMode = (mode === 'edit');
    
    document.getElementById('edit-path').value = tabData.filePath;
    document.getElementById('edit-copy').value = (mode === 'upload') ? "true" : "false";
    document.getElementById('edit-title').value = tabData.title;
    document.getElementById('edit-artist').value = tabData.artist;
    document.getElementById('edit-album').value = tabData.album;
    document.getElementById('edit-type').value = tabData.type;
    
    // Country & Lang
    document.getElementById('edit-country').value = tabData.country || "US";
    document.getElementById('edit-lang').value = tabData.language || "en_us";

    document.getElementById('edit-form').dataset.id = tabData.id;

    const titleEl = document.querySelector('#modal-overlay .modal h2');
    if (isEditMode) titleEl.innerText = "Edit Tab Metadata";
    else titleEl.innerText = "Add New Tab";

    document.getElementById('modal-overlay').classList.remove('hidden');
}

function closeModal() {
    document.getElementById('modal-overlay').classList.add('hidden');
    document.getElementById('edit-form').reset();
}

async function saveTab() {
    const id = document.getElementById('edit-form').dataset.id;
    const path = document.getElementById('edit-path').value;
    const shouldCopy = document.getElementById('edit-copy').value === "true";
    const title = document.getElementById('edit-title').value;
    const artist = document.getElementById('edit-artist').value;
    const album = document.getElementById('edit-album').value;
    const type = document.getElementById('edit-type').value;
    const country = document.getElementById('edit-country').value;
    const lang = document.getElementById('edit-lang').value;

    let existingTab = tabs.find(t => t.id === id);
    
    const tab = {
        id: id,
        title: title,
        artist: artist,
        album: album,
        filePath: path,
        type: type,
        isManaged: existingTab ? existingTab.isManaged : false,
        coverPath: existingTab ? existingTab.coverPath : "",
        categoryId: existingTab ? existingTab.categoryId : currentCategoryId, // Default to current cat if new
        country: country,
        language: lang
    };

    try {
        if (isEditMode) {
            await window.go.main.App.UpdateTab(tab);
            showToast("Tab updated. Fetching cover...");
        } else {
            await window.go.main.App.SaveTab(tab, shouldCopy);
            showToast("Tab added.");
        }
        closeModal();
        refreshData();
    } catch (err) {
        alert("Error saving tab: " + err);
    }
}

async function openTab(id) {
    try {
        await window.go.main.App.OpenTab(id);
    } catch (err) {
        console.error(err);
        showToast("Failed to open tab", "error");
    }
}

async function deleteTab(id) {
    if(confirm("Are you sure you want to delete/unlink this tab?")) {
        try {
            await window.go.main.App.DeleteTab(id);
            refreshData();
            showToast("Tab deleted");
        } catch(e) {
            alert("Error: " + e);
        }
    }
}

async function exportTab(id) {
    const dest = await window.go.main.App.SelectFolder();
    if(dest) {
        try {
            await window.go.main.App.ExportTab(id, dest);
            showToast("Exported successfully!");
        } catch(e) {
            alert("Export failed: " + e);
        }
    }
}

// --- Category Management ---

function openCategoryModal(cat = null) {
    document.getElementById('category-modal').classList.remove('hidden');
    const titleEl = document.querySelector('#category-modal h2');
    const submitBtn = document.querySelector('#category-modal button[type="submit"]');

    if (cat) {
        // Edit Mode
        document.getElementById('cat-id').value = cat.id;
        document.getElementById('cat-name').value = cat.name;
        titleEl.innerText = "Rename Category";
        submitBtn.innerText = "Save";
    } else {
        // Create Mode
        document.getElementById('cat-id').value = "";
        document.getElementById('cat-name').value = "";
        titleEl.innerText = "New Category";
        submitBtn.innerText = "Create";
    }
    document.getElementById('cat-name').focus();
}

function closeCategoryModal() {
    document.getElementById('category-modal').classList.add('hidden');
}

async function saveCategory() {
    const id = document.getElementById('cat-id').value;
    const name = document.getElementById('cat-name').value;
    if(!name) return;

    const cat = {
        id: id, // If empty, backend generates new ID
        name: name,
        parentId: id ? categories.find(c => c.id === id).parentId : currentCategoryId
    };
    
    try {
        await window.go.main.App.AddCategory(cat);
        closeCategoryModal();
        refreshData();
        showToast(id ? "Category renamed" : "Category created");
    } catch(e) {
        alert("Error saving category: " + e);
    }
}

async function deleteCategory(id) {
    if(confirm("Delete this category? content inside will be moved to root.")) {
        await window.go.main.App.DeleteCategory(id);
        refreshData();
    }
}

// --- Move Tab Logic ---

function promptMoveTab(tabId) {
    const modal = document.getElementById('move-modal');
    const select = document.getElementById('move-select');
    const idInput = document.getElementById('move-tab-id');

    // Populate select
    select.innerHTML = '<option value="">(Root)</option>';
    
    // Sort categories by name for better display? Or just list them.
    // Let's create a map or just iterate.
    const options = categories.map(c => {
        return {
            id: c.id,
            name: getCategoryPath(c.id)
        };
    });
    
    // Sort alphabetically
    options.sort((a, b) => a.name.localeCompare(b.name));

    options.forEach(opt => {
        const option = document.createElement('option');
        option.value = opt.id;
        option.innerText = opt.name;
        select.appendChild(option);
    });

    idInput.value = tabId;
    modal.classList.remove('hidden');
}

function getCategoryPath(id) {
    let parts = [];
    let current = categories.find(c => c.id === id);
    while(current) {
        parts.unshift(current.name);
        current = categories.find(c => c.id === current.parentId);
    }
    return parts.join(" / ");
}

async function saveMove() {
    const tabId = document.getElementById('move-tab-id').value;
    const catId = document.getElementById('move-select').value;
    
    try {
        await window.go.main.App.MoveTab(tabId, catId);
        document.getElementById('move-modal').classList.add('hidden');
        refreshData();
        showToast("Tab moved");
    } catch(e) {
        alert("Error moving tab: " + e);
    }
}

// --- Toast ---
function showToast(message, type = "info") {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.innerText = message;
    
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}