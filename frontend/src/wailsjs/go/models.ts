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
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parentId = source["parentId"];
	    }
	}
	export class Settings {
	    theme: string;
	    background: string;
	    bgType: string;
	    openMethod: string;
	    openGpMethod: string;
	    syncPaths: string[];
	    syncStrategy: string;
	
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
	        this.syncPaths = source["syncPaths"];
	        this.syncStrategy = source["syncStrategy"];
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
	    categoryId: string;
	    country: string;
	    language: string;
	    tag: string;
	
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
	        this.categoryId = source["categoryId"];
	        this.country = source["country"];
	        this.language = source["language"];
	        this.tag = source["tag"];
	    }
	}

}

