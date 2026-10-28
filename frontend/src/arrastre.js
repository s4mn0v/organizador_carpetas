import {ref} from 'vue'
import {padreDe} from './util'

// Arrastrar y soltar en la barra lateral (compartido entre todos los niveles del árbol).
export const arrastrado = ref(null)   // entrada que se está arrastrando
export const sobre = ref(null)        // ruta de la carpeta bajo el cursor

// Se puede soltar en una carpeta que no sea ella misma, ni una de sus subcarpetas, ni donde ya está.
export function sePuedeSoltar(destino) {
  const o = arrastrado.value
  if (!o || !destino?.esCarpeta || !destino.ruta) return false
  if (destino.ruta === o.ruta || destino.ruta.startsWith(o.ruta + '/')) return false
  return padreDe(o.ruta) !== destino.ruta
}
export function terminarArrastre() {
  arrastrado.value = null
  sobre.value = null
}
