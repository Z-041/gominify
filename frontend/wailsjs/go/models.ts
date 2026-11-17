export namespace history {
	
	export class Entry {
	    time: string;
	    fileCount: number;
	    failed: number;
	    originalTotal: number;
	    minifiedTotal: number;
	    destDir: string;
	    jsMangle: boolean;
	    keepCopyright: boolean;
	    cssPrecision: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.fileCount = source["fileCount"];
	        this.failed = source["failed"];
	        this.originalTotal = source["originalTotal"];
	        this.minifiedTotal = source["minifiedTotal"];
	        this.destDir = source["destDir"];
	        this.jsMangle = source["jsMangle"];
	        this.keepCopyright = source["keepCopyright"];
	        this.cssPrecision = source["cssPrecision"];
	    }
	}

}

export namespace main {
	
	export class FileResult {
	    id: string;
	    path: string;
	    rootPath: string;
	    relPath: string;
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
	        this.rootPath = source["rootPath"];
	        this.relPath = source["relPath"];
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
	export class BatchResult {
	    results: FileResult[];
	    skipped: number;
	    warnings: string[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], FileResult);
	        this.skipped = source["skipped"];
	        this.warnings = source["warnings"];
	        this.truncated = source["truncated"];
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

