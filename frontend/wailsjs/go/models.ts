export namespace main {
	
	export class FileResult {
	    id: string;
	    path: string;
	    name: string;
	    language: string;
	    originalSize: number;
	    minifiedSize: number;
	    ratio: number;
	    elapsedMs: number;
	    output: string;
	    source: string;
	    error: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new FileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.language = source["language"];
	        this.originalSize = source["originalSize"];
	        this.minifiedSize = source["minifiedSize"];
	        this.ratio = source["ratio"];
	        this.elapsedMs = source["elapsedMs"];
	        this.output = source["output"];
	        this.source = source["source"];
	        this.error = source["error"];
	        this.line = source["line"];
	    }
	}
	export class UpdateInfo {
	    available: boolean;
	    current: string;
	    latest: string;
	    notes: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.notes = source["notes"];
	        this.size = source["size"];
	    }
	}

}

export namespace minifier {
	
	export class Options {
	    JSMangle: boolean;
	    KeepCopyright: boolean;
	    CSSPrecision: number;
	
	    static createFrom(source: any = {}) {
	        return new Options(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.JSMangle = source["JSMangle"];
	        this.KeepCopyright = source["KeepCopyright"];
	        this.CSSPrecision = source["CSSPrecision"];
	    }
	}

}

