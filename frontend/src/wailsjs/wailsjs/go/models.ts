export namespace main {
	
	export class TabsResponse {
	    tabs: store.Tab[];
	    total: number;
	    page: number;
	    pageSize: number;
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TabsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tabs = this.convertValues(source["tabs"], store.Tab);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.hasMore = source["hasMore"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace store {
	
	export class Category {
	    id: string;
	    name: string;
	    parentId: string;
	    coverPath: string;
	    effectiveCoverPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parentId = source["parentId"];
	        this.coverPath = source["coverPath"];
	        this.effectiveCoverPath = source["effectiveCoverPath"];
	    }
	}
	export class KeyBindings {
	    scrollDown: string;
	    scrollUp: string;
	    metronome: string;
	    playPause: string;
	    stop: string;
	    bpmPlus: string;
	    bpmMinus: string;
	    toggleLoop: string;
	    clearSelection: string;
	    jumpToBar: string;
	    jumpToStart: string;
	    autoScroll: string;
	    scrollSpeedUp: string;
	    scrollSpeedDown: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyBindings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scrollDown = source["scrollDown"];
	        this.scrollUp = source["scrollUp"];
	        this.metronome = source["metronome"];
	        this.playPause = source["playPause"];
	        this.stop = source["stop"];
	        this.bpmPlus = source["bpmPlus"];
	        this.bpmMinus = source["bpmMinus"];
	        this.toggleLoop = source["toggleLoop"];
	        this.clearSelection = source["clearSelection"];
	        this.jumpToBar = source["jumpToBar"];
	        this.jumpToStart = source["jumpToStart"];
	        this.autoScroll = source["autoScroll"];
	        this.scrollSpeedUp = source["scrollSpeedUp"];
	        this.scrollSpeedDown = source["scrollSpeedDown"];
	    }
	}
	export class Settings {
	    theme: string;
	    background: string;
	    bgType: string;
	    openMethod: string;
	    openGpMethod: string;
	    audioDevice: string;
	    syncPaths: string[];
	    syncStrategy: string;
	    autoSyncEnabled: boolean;
	    autoSyncFrequency: string;
	    lastSyncTime: number;
	    keyBindings: KeyBindings;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.background = source["background"];
	        this.bgType = source["bgType"];
	        this.openMethod = source["openMethod"];
	        this.openGpMethod = source["openGpMethod"];
	        this.audioDevice = source["audioDevice"];
	        this.syncPaths = source["syncPaths"];
	        this.syncStrategy = source["syncStrategy"];
	        this.autoSyncEnabled = source["autoSyncEnabled"];
	        this.autoSyncFrequency = source["autoSyncFrequency"];
	        this.lastSyncTime = source["lastSyncTime"];
	        this.keyBindings = this.convertValues(source["keyBindings"], KeyBindings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Tab {
	    id: string;
	    title: string;
	    artist: string;
	    album: string;
	    filePath: string;
	    type: string;
	    isManaged: boolean;
	    coverPath: string;
	    categoryIds: string[];
	    country: string;
	    language: string;
	    tag: string;
	    addedAt: number;
	    lastOpened: number;
	
	    static createFrom(source: any = {}) {
	        return new Tab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.filePath = source["filePath"];
	        this.type = source["type"];
	        this.isManaged = source["isManaged"];
	        this.coverPath = source["coverPath"];
	        this.categoryIds = source["categoryIds"];
	        this.country = source["country"];
	        this.language = source["language"];
	        this.tag = source["tag"];
	        this.addedAt = source["addedAt"];
	        this.lastOpened = source["lastOpened"];
	    }
	}

}

