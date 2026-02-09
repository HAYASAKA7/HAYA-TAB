export namespace store {
	
	export class Tab {
	    id: string;
	    title: string;
	    artist: string;
	    album: string;
	    filePath: string;
	    type: string;
	    isManaged: boolean;
	    coverPath: string;
	
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
	    }
	}

}

