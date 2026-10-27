export namespace main {
	
	export class Campo {
	    nombre: string;
	    grupo: string;
	    tipo: string;
	    opciones: string[];
	    orden: number;
	
	    static createFrom(source: any = {}) {
	        return new Campo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nombre = source["nombre"];
	        this.grupo = source["grupo"];
	        this.tipo = source["tipo"];
	        this.opciones = source["opciones"];
	        this.orden = source["orden"];
	    }
	}
	export class DescripcionSub {
	    ruta: string;
	    descripcion: string;
	    actualizado: string;
	
	    static createFrom(source: any = {}) {
	        return new DescripcionSub(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruta = source["ruta"];
	        this.descripcion = source["descripcion"];
	        this.actualizado = source["actualizado"];
	    }
	}
	export class Entrada {
	    nombre: string;
	    ruta: string;
	    esCarpeta: boolean;
	    ext: string;
	    tamano: number;
	    modificado: string;
	    ficha: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entrada(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nombre = source["nombre"];
	        this.ruta = source["ruta"];
	        this.esCarpeta = source["esCarpeta"];
	        this.ext = source["ext"];
	        this.tamano = source["tamano"];
	        this.modificado = source["modificado"];
	        this.ficha = source["ficha"];
	    }
	}
	export class DetalleArchivo {
	    archivo: Entrada;
	    creado: string;
	
	    static createFrom(source: any = {}) {
	        return new DetalleArchivo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.archivo = this.convertValues(source["archivo"], Entrada);
	        this.creado = source["creado"];
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
	export class SubcarpetaFicha {
	    ruta: string;
	    nombre: string;
	    nivel: number;
	    descripcion: string;
	
	    static createFrom(source: any = {}) {
	        return new SubcarpetaFicha(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruta = source["ruta"];
	        this.nombre = source["nombre"];
	        this.nivel = source["nivel"];
	        this.descripcion = source["descripcion"];
	    }
	}
	export class Registro {
	    id: number;
	    ruta: string;
	    actualizado: string;
	    valores: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Registro(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ruta = source["ruta"];
	        this.actualizado = source["actualizado"];
	        this.valores = source["valores"];
	    }
	}
	export class DetalleCarpeta {
	    carpeta: Entrada;
	    entradas: Entrada[];
	    subcarpetas: number;
	    archivos: number;
	    tamanoTotal: number;
	    registros: Registro[];
	    esPrincipal: boolean;
	    arbol: SubcarpetaFicha[];
	    arbolCortado: boolean;
	    principal: string;
	    principalFicha: boolean;
	    descripcion: string;
	
	    static createFrom(source: any = {}) {
	        return new DetalleCarpeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.carpeta = this.convertValues(source["carpeta"], Entrada);
	        this.entradas = this.convertValues(source["entradas"], Entrada);
	        this.subcarpetas = source["subcarpetas"];
	        this.archivos = source["archivos"];
	        this.tamanoTotal = source["tamanoTotal"];
	        this.registros = this.convertValues(source["registros"], Registro);
	        this.esPrincipal = source["esPrincipal"];
	        this.arbol = this.convertValues(source["arbol"], SubcarpetaFicha);
	        this.arbolCortado = source["arbolCortado"];
	        this.principal = source["principal"];
	        this.principalFicha = source["principalFicha"];
	        this.descripcion = source["descripcion"];
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
	export class EnlaceGrafo {
	    origen: string;
	    destino: string;
	
	    static createFrom(source: any = {}) {
	        return new EnlaceGrafo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.origen = source["origen"];
	        this.destino = source["destino"];
	    }
	}
	export class Estado {
	    raiz: string;
	    nombreRaiz: string;
	    recientes: string[];
	    base: string;
	    baseExiste: boolean;
	    baseError: string;
	    campos: Campo[];
	    registros: number;
	    sinCarpeta: Registro[];
	    noPrincipal: Registro[];
	    papeleraHay: boolean;
	    nombreExcel: string;
	    nombrePapelera: string;
	
	    static createFrom(source: any = {}) {
	        return new Estado(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.raiz = source["raiz"];
	        this.nombreRaiz = source["nombreRaiz"];
	        this.recientes = source["recientes"];
	        this.base = source["base"];
	        this.baseExiste = source["baseExiste"];
	        this.baseError = source["baseError"];
	        this.campos = this.convertValues(source["campos"], Campo);
	        this.registros = source["registros"];
	        this.sinCarpeta = this.convertValues(source["sinCarpeta"], Registro);
	        this.noPrincipal = this.convertValues(source["noPrincipal"], Registro);
	        this.papeleraHay = source["papeleraHay"];
	        this.nombreExcel = source["nombreExcel"];
	        this.nombrePapelera = source["nombrePapelera"];
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
	export class NodoGrafo {
	    id: string;
	    nombre: string;
	    esCarpeta: boolean;
	    ext: string;
	    nivel: number;
	    hijos: number;
	    ficha: boolean;
	    tamano: number;
	
	    static createFrom(source: any = {}) {
	        return new NodoGrafo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nombre = source["nombre"];
	        this.esCarpeta = source["esCarpeta"];
	        this.ext = source["ext"];
	        this.nivel = source["nivel"];
	        this.hijos = source["hijos"];
	        this.ficha = source["ficha"];
	        this.tamano = source["tamano"];
	    }
	}
	export class Grafo {
	    nodos: NodoGrafo[];
	    enlaces: EnlaceGrafo[];
	    cortado: boolean;
	    maximo: number;
	
	    static createFrom(source: any = {}) {
	        return new Grafo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodos = this.convertValues(source["nodos"], NodoGrafo);
	        this.enlaces = this.convertValues(source["enlaces"], EnlaceGrafo);
	        this.cortado = source["cortado"];
	        this.maximo = source["maximo"];
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
	export class ResultadoBusqueda {
	    entradas: Entrada[];
	    cortada: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResultadoBusqueda(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entradas = this.convertValues(source["entradas"], Entrada);
	        this.cortada = source["cortada"];
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
	export class ResumenEliminar {
	    carpetas: number;
	    archivos: number;
	    fichas: number;
	    descripciones: number;
	    cortado: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResumenEliminar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.carpetas = source["carpetas"];
	        this.archivos = source["archivos"];
	        this.fichas = source["fichas"];
	        this.descripciones = source["descripciones"];
	        this.cortado = source["cortado"];
	    }
	}

}

