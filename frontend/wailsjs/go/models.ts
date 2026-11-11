export namespace main {
	
	export class ArchivoVencido {
	    ruta: string;
	    nombre: string;
	    ext: string;
	    carpeta: string;
	    tamano: number;
	    modificado: string;
	    creado: string;
	    retencion: string;
	    origen: string;
	    vencio: string;
	    dias: number;
	
	    static createFrom(source: any = {}) {
	        return new ArchivoVencido(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruta = source["ruta"];
	        this.nombre = source["nombre"];
	        this.ext = source["ext"];
	        this.carpeta = source["carpeta"];
	        this.tamano = source["tamano"];
	        this.modificado = source["modificado"];
	        this.creado = source["creado"];
	        this.retencion = source["retencion"];
	        this.origen = source["origen"];
	        this.vencio = source["vencio"];
	        this.dias = source["dias"];
	    }
	}
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
	export class UsoClaude {
	    entrada: number;
	    salida: number;
	    cacheLectura: number;
	    cacheCreacion: number;
	    costoUSD: number;
	    turnos: number;
	    duracionMs: number;
	    mensajes: number;
	
	    static createFrom(source: any = {}) {
	        return new UsoClaude(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entrada = source["entrada"];
	        this.salida = source["salida"];
	        this.cacheLectura = source["cacheLectura"];
	        this.cacheCreacion = source["cacheCreacion"];
	        this.costoUSD = source["costoUSD"];
	        this.turnos = source["turnos"];
	        this.duracionMs = source["duracionMs"];
	        this.mensajes = source["mensajes"];
	    }
	}
	export class Conversacion {
	    id: string;
	    titulo: string;
	    sesion: string;
	    creada: string;
	    actualizada: string;
	    mensajes: string;
	    uso: UsoClaude;
	
	    static createFrom(source: any = {}) {
	        return new Conversacion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.titulo = source["titulo"];
	        this.sesion = source["sesion"];
	        this.creada = source["creada"];
	        this.actualizada = source["actualizada"];
	        this.mensajes = source["mensajes"];
	        this.uso = this.convertValues(source["uso"], UsoClaude);
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
	export class ConversacionClaude {
	    conv: string;
	    sesion: string;
	    ocupado: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConversacionClaude(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conv = source["conv"];
	        this.sesion = source["sesion"];
	        this.ocupado = source["ocupado"];
	    }
	}
	export class CopiaDup {
	    ruta: string;
	    carpeta: string;
	    ubicacion: string;
	    tamano: number;
	    creado: string;
	    modificado: string;
	
	    static createFrom(source: any = {}) {
	        return new CopiaDup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruta = source["ruta"];
	        this.carpeta = source["carpeta"];
	        this.ubicacion = source["ubicacion"];
	        this.tamano = source["tamano"];
	        this.creado = source["creado"];
	        this.modificado = source["modificado"];
	    }
	}
	export class DescripcionSub {
	    ruta: string;
	    descripcion: string;
	    actualizado: string;
	    retencion: string;
	
	    static createFrom(source: any = {}) {
	        return new DescripcionSub(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruta = source["ruta"];
	        this.descripcion = source["descripcion"];
	        this.actualizado = source["actualizado"];
	        this.retencion = source["retencion"];
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
	    ubicacion: string;
	    retencion: string;
	    retencionDe: string;
	    vence: string;
	
	    static createFrom(source: any = {}) {
	        return new DetalleArchivo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.archivo = this.convertValues(source["archivo"], Entrada);
	        this.creado = source["creado"];
	        this.ubicacion = source["ubicacion"];
	        this.retencion = source["retencion"];
	        this.retencionDe = source["retencionDe"];
	        this.vence = source["vence"];
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
	    retencion: string;
	    heredada: string;
	
	    static createFrom(source: any = {}) {
	        return new SubcarpetaFicha(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruta = source["ruta"];
	        this.nombre = source["nombre"];
	        this.nivel = source["nivel"];
	        this.descripcion = source["descripcion"];
	        this.retencion = source["retencion"];
	        this.heredada = source["heredada"];
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
	    esInicio: boolean;
	    esRaiz: boolean;
	    esPrincipal: boolean;
	    arbol: SubcarpetaFicha[];
	    arbolCortado: boolean;
	    principal: string;
	    principalFicha: boolean;
	    descripcion: string;
	    retencion: string;
	    retencionDe: string;
	
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
	        this.esInicio = source["esInicio"];
	        this.esRaiz = source["esRaiz"];
	        this.esPrincipal = source["esPrincipal"];
	        this.arbol = this.convertValues(source["arbol"], SubcarpetaFicha);
	        this.arbolCortado = source["arbolCortado"];
	        this.principal = source["principal"];
	        this.principalFicha = source["principalFicha"];
	        this.descripcion = source["descripcion"];
	        this.retencion = source["retencion"];
	        this.retencionDe = source["retencionDe"];
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
	
	export class EstadoRaiz {
	    id: string;
	    ruta: string;
	    nombre: string;
	    color: string;
	    accesible: boolean;
	    error: string;
	    base: string;
	    baseExiste: boolean;
	    baseError: string;
	    campos: Campo[];
	    registros: number;
	    sinCarpeta: Registro[];
	    noPrincipal: Registro[];
	    papeleraHay: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EstadoRaiz(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ruta = source["ruta"];
	        this.nombre = source["nombre"];
	        this.color = source["color"];
	        this.accesible = source["accesible"];
	        this.error = source["error"];
	        this.base = source["base"];
	        this.baseExiste = source["baseExiste"];
	        this.baseError = source["baseError"];
	        this.campos = this.convertValues(source["campos"], Campo);
	        this.registros = source["registros"];
	        this.sinCarpeta = this.convertValues(source["sinCarpeta"], Registro);
	        this.noPrincipal = this.convertValues(source["noPrincipal"], Registro);
	        this.papeleraHay = source["papeleraHay"];
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
	export class Estado {
	    raices: EstadoRaiz[];
	    recientes: string[];
	    paleta: string[];
	    nombreExcel: string;
	    nombrePapelera: string;
	    avisos: string[];
	
	    static createFrom(source: any = {}) {
	        return new Estado(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.raices = this.convertValues(source["raices"], EstadoRaiz);
	        this.recientes = source["recientes"];
	        this.paleta = source["paleta"];
	        this.nombreExcel = source["nombreExcel"];
	        this.nombrePapelera = source["nombrePapelera"];
	        this.avisos = source["avisos"];
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
	export class EstadoClaude {
	    activo: boolean;
	    url: string;
	    config: string;
	
	    static createFrom(source: any = {}) {
	        return new EstadoClaude(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activo = source["activo"];
	        this.url = source["url"];
	        this.config = source["config"];
	    }
	}
	export class EstadoIndice {
	    raiz: string;
	    listo: boolean;
	    cargando: boolean;
	    carpetas: number;
	    archivos: number;
	    leidos: number;
	    vigilando: boolean;
	    dudoso: boolean;
	    generado: string;
	    parcial: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EstadoIndice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.raiz = source["raiz"];
	        this.listo = source["listo"];
	        this.cargando = source["cargando"];
	        this.carpetas = source["carpetas"];
	        this.archivos = source["archivos"];
	        this.leidos = source["leidos"];
	        this.vigilando = source["vigilando"];
	        this.dudoso = source["dudoso"];
	        this.generado = source["generado"];
	        this.parcial = source["parcial"];
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
	export class GrupoDup {
	    nombre: string;
	    ext: string;
	    tamano: number;
	    huella: string;
	    copias: CopiaDup[];
	    desperdicio: number;
	
	    static createFrom(source: any = {}) {
	        return new GrupoDup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nombre = source["nombre"];
	        this.ext = source["ext"];
	        this.tamano = source["tamano"];
	        this.huella = source["huella"];
	        this.copias = this.convertValues(source["copias"], CopiaDup);
	        this.desperdicio = source["desperdicio"];
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
	export class HerramientaClaude {
	    nombre: string;
	    etiqueta: string;
	    tipo: string;
	    pista: string;
	
	    static createFrom(source: any = {}) {
	        return new HerramientaClaude(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nombre = source["nombre"];
	        this.etiqueta = source["etiqueta"];
	        this.tipo = source["tipo"];
	        this.pista = source["pista"];
	    }
	}
	
	export class Raiz {
	    id: string;
	    ruta: string;
	    nombre: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new Raiz(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ruta = source["ruta"];
	        this.nombre = source["nombre"];
	        this.color = source["color"];
	    }
	}
	
	export class ResultadoBusqueda {
	    entradas: Entrada[];
	    total: number;
	    cortada: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResultadoBusqueda(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entradas = this.convertValues(source["entradas"], Entrada);
	        this.total = source["total"];
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
	export class ResultadoDepurar {
	    movidas: string[];
	    movidos: number;
	    tamano: number;
	    errores: string[];
	
	    static createFrom(source: any = {}) {
	        return new ResultadoDepurar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.movidas = source["movidas"];
	        this.movidos = source["movidos"];
	        this.tamano = source["tamano"];
	        this.errores = source["errores"];
	    }
	}
	export class ResultadoDuplicados {
	    grupos: GrupoDup[];
	    revisados: number;
	    comparados: number;
	    desperdicio: number;
	    cancelado: boolean;
	    errores: number;
	    segundos: number;
	
	    static createFrom(source: any = {}) {
	        return new ResultadoDuplicados(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.grupos = this.convertValues(source["grupos"], GrupoDup);
	        this.revisados = source["revisados"];
	        this.comparados = source["comparados"];
	        this.desperdicio = source["desperdicio"];
	        this.cancelado = source["cancelado"];
	        this.errores = source["errores"];
	        this.segundos = source["segundos"];
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
	export class ResultadoSubir {
	    carpetas: string[];
	    archivos: number;
	    tamano: number;
	    errores: string[];
	    cancelado: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResultadoSubir(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.carpetas = source["carpetas"];
	        this.archivos = source["archivos"];
	        this.tamano = source["tamano"];
	        this.errores = source["errores"];
	        this.cancelado = source["cancelado"];
	    }
	}
	export class ResultadoVencidos {
	    archivos: ArchivoVencido[];
	    revisados: number;
	    reglas: number;
	    sinExcel: string[];
	    cortado: boolean;
	    cancelado: boolean;
	    segundos: number;
	
	    static createFrom(source: any = {}) {
	        return new ResultadoVencidos(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.archivos = this.convertValues(source["archivos"], ArchivoVencido);
	        this.revisados = source["revisados"];
	        this.reglas = source["reglas"];
	        this.sinExcel = source["sinExcel"];
	        this.cortado = source["cortado"];
	        this.cancelado = source["cancelado"];
	        this.segundos = source["segundos"];
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
	export class ResumenConversacion {
	    id: string;
	    titulo: string;
	    actualizada: string;
	    actual: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResumenConversacion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.titulo = source["titulo"];
	        this.actualizada = source["actualizada"];
	        this.actual = source["actual"];
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
	
	export class TamanoTotal {
	    ruta: string;
	    tamano: number;
	    archivos: number;
	    carpetas: number;
	    cargando: boolean;
	    enCola: boolean;
	    segundos: number;
	    hora: string;
	
	    static createFrom(source: any = {}) {
	        return new TamanoTotal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruta = source["ruta"];
	        this.tamano = source["tamano"];
	        this.archivos = source["archivos"];
	        this.carpetas = source["carpetas"];
	        this.cargando = source["cargando"];
	        this.enCola = source["enCola"];
	        this.segundos = source["segundos"];
	        this.hora = source["hora"];
	    }
	}

}

