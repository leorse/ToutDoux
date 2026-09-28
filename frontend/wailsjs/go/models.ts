export namespace domain {
	
	export class Meeting {
	    id: string;
	    projectId: string;
	    title: string;
	    hidden: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Meeting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.title = source["title"];
	        this.hidden = source["hidden"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class MeetingInstance {
	    id: string;
	    meetingId: string;
	    notes: string;
	    // Go type: time
	    timestamp: any;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new MeetingInstance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.meetingId = source["meetingId"];
	        this.notes = source["notes"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class Note {
	    id: string;
	    projectId: string;
	    title: string;
	    content: string;
	    hidden: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.hidden = source["hidden"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class Project {
	    id: string;
	    name: string;
	    locked: boolean;
	    hidden: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.locked = source["locked"];
	        this.hidden = source["hidden"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class Task {
	    id: string;
	    projectId: string;
	    parentId?: string;
	    name: string;
	    description: string;
	    importance: string;
	    completed: boolean;
	    cancelled: boolean;
	    // Go type: time
	    dueDate?: any;
	    orderIndex: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.parentId = source["parentId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.importance = source["importance"];
	        this.completed = source["completed"];
	        this.cancelled = source["cancelled"];
	        this.dueDate = this.convertValues(source["dueDate"], null);
	        this.orderIndex = source["orderIndex"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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

export namespace duedate {
	
	export class Info {
	    text: string;
	    urgent: boolean;
	    overdue: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.urgent = source["urgent"];
	        this.overdue = source["overdue"];
	    }
	}

}

export namespace main {
	
	export class Release {
	    version: string;
	    date: string;
	    important?: string;
	    changes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Release(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.date = source["date"];
	        this.important = source["important"];
	        this.changes = source["changes"];
	    }
	}
	export class AppInfo {
	    version: string;
	    releases: Release[];
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.releases = this.convertValues(source["releases"], Release);
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
	export class CreateTaskResult {
	    task: domain.Task;
	    reactivated: domain.Task[];
	
	    static createFrom(source: any = {}) {
	        return new CreateTaskResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task = this.convertValues(source["task"], domain.Task);
	        this.reactivated = this.convertValues(source["reactivated"], domain.Task);
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
	export class DeleteProjectSummary {
	    tasks: number;
	    notes: number;
	    meetings: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteProjectSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tasks = source["tasks"];
	        this.notes = source["notes"];
	        this.meetings = source["meetings"];
	    }
	}
	export class FilterSelection {
	    status: string[];
	    importance: string[];
	    dueOnly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FilterSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.importance = source["importance"];
	        this.dueOnly = source["dueOnly"];
	    }
	}
	
	export class SearchResult {
	    type: string;
	    id: string;
	    title: string;
	    projectId: string;
	    // Go type: time
	    date?: any;
	    task?: domain.Task;
	    note?: domain.Note;
	    meeting?: domain.Meeting;
	    instance?: domain.MeetingInstance;
	    project?: domain.Project;
	    snippet?: search.Snippet;
	    projectName: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.projectId = source["projectId"];
	        this.date = this.convertValues(source["date"], null);
	        this.task = this.convertValues(source["task"], domain.Task);
	        this.note = this.convertValues(source["note"], domain.Note);
	        this.meeting = this.convertValues(source["meeting"], domain.Meeting);
	        this.instance = this.convertValues(source["instance"], domain.MeetingInstance);
	        this.project = this.convertValues(source["project"], domain.Project);
	        this.snippet = this.convertValues(source["snippet"], search.Snippet);
	        this.projectName = source["projectName"];
	        this.score = source["score"];
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
	export class SemanticStatus {
	    modelAvailable: boolean;
	    filesPresent: boolean;
	    indexedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SemanticStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelAvailable = source["modelAvailable"];
	        this.filesPresent = source["filesPresent"];
	        this.indexedCount = source["indexedCount"];
	    }
	}
	export class TaskPatch {
	    name?: string;
	    description?: string;
	    importance?: string;
	    dueDate?: string;
	    clearDue: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TaskPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.importance = source["importance"];
	        this.dueDate = source["dueDate"];
	        this.clearDue = source["clearDue"];
	    }
	}
	export class TaskView {
	    tasks: domain.Task[];
	    visible: string[];
	    defaultExpanded: string[];
	    due: Record<string, duedate.Info>;
	
	    static createFrom(source: any = {}) {
	        return new TaskView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tasks = this.convertValues(source["tasks"], domain.Task);
	        this.visible = source["visible"];
	        this.defaultExpanded = source["defaultExpanded"];
	        this.due = this.convertValues(source["due"], duedate.Info, true);
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

export namespace model {
	
	export class File {
	    name: string;
	    source: string;
	    present: boolean;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.source = source["source"];
	        this.present = source["present"];
	        this.size = source["size"];
	    }
	}
	export class Status {
	    available: boolean;
	    directory: string;
	    expectedFiles: string[];
	    missingFiles: string[];
	    downloadUrl: string;
	    files: File[];
	    runtime: File;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.directory = source["directory"];
	        this.expectedFiles = source["expectedFiles"];
	        this.missingFiles = source["missingFiles"];
	        this.downloadUrl = source["downloadUrl"];
	        this.files = this.convertValues(source["files"], File);
	        this.runtime = this.convertValues(source["runtime"], File);
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

export namespace search {
	
	export class Part {
	    text: string;
	    match: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Part(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.match = source["match"];
	    }
	}
	export class Snippet {
	    leadingEllipsis: boolean;
	    parts: Part[];
	    trailingEllipsis: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Snippet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.leadingEllipsis = source["leadingEllipsis"];
	        this.parts = this.convertValues(source["parts"], Part);
	        this.trailingEllipsis = source["trailingEllipsis"];
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

export namespace stats {
	
	export class PriorityTasks {
	    criticalOrHigh: domain.Task[];
	    dueSoon: domain.Task[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new PriorityTasks(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.criticalOrHigh = this.convertValues(source["criticalOrHigh"], domain.Task);
	        this.dueSoon = this.convertValues(source["dueSoon"], domain.Task);
	        this.total = source["total"];
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
	export class SidebarStats {
	    activeCount: number;
	    hasCritical: boolean;
	    hasHigh: boolean;
	    dueIcon: string;
	    notesCount: number;
	    meetingsCount: number;
	    hiddenNotesCount: number;
	    hiddenMeetingsCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SidebarStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeCount = source["activeCount"];
	        this.hasCritical = source["hasCritical"];
	        this.hasHigh = source["hasHigh"];
	        this.dueIcon = source["dueIcon"];
	        this.notesCount = source["notesCount"];
	        this.meetingsCount = source["meetingsCount"];
	        this.hiddenNotesCount = source["hiddenNotesCount"];
	        this.hiddenMeetingsCount = source["hiddenMeetingsCount"];
	    }
	}

}

