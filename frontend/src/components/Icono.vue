<script setup>
defineProps({nombre: {type: String, required: true}})
</script>

<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <template v-if="nombre === 'chevron'"><path d="m9 6 6 6-6 6"/></template>
    <template v-else-if="nombre === 'buscar'"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></template>
    <template v-else-if="nombre === 'carpeta'"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></template>
    <template v-else-if="nombre === 'abrir'"><path d="M14 4h6v6"/><path d="M20 4 11 13"/><path d="M19 14v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4"/></template>
    <template v-else-if="nombre === 'borrar'"><path d="M4 7h16"/><path d="M10 11v6M14 11v6"/><path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12"/><path d="M9 7V4h6v3"/></template>
    <template v-else-if="nombre === 'editar'"><path d="M4 20h4L19 9l-4-4L4 16z"/><path d="m13 7 4 4"/></template>
    <template v-else-if="nombre === 'carpeta-mas'"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M12 10v6M9 13h6"/></template>
    <template v-else-if="nombre === 'mover'"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M10 13h6m-2-2 2 2-2 2"/></template>
    <template v-else-if="nombre === 'sync'"><path d="M20 11a8 8 0 0 0-14.6-4.5L4 8"/><path d="M4 4v4h4"/><path d="M4 13a8 8 0 0 0 14.6 4.5L20 16"/><path d="M20 20v-4h-4"/></template>
    <template v-else-if="nombre === 'excel'"><rect x="4" y="3" width="16" height="18" rx="2"/><path d="M8 8h8M8 12h8M8 16h5"/></template>
    <template v-else-if="nombre === 'mas'"><path d="M12 5v14M5 12h14"/></template>
    <template v-else-if="nombre === 'alerta'"><path d="M12 3 2 20h20z"/><path d="M12 10v4M12 17h.01"/></template>
    <template v-else-if="nombre === 'cerrar'"><path d="M6 6l12 12M18 6 6 18"/></template>
    <template v-else-if="nombre === 'recargar'"><path d="M20 12a8 8 0 1 1-2.3-5.7L20 8"/><path d="M20 4v4h-4"/></template>
    <template v-else-if="nombre === 'subir'"><path d="M12 15V4"/><path d="m7 9 5-5 5 5"/><path d="M5 15v3a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-3"/></template>
    <template v-else-if="nombre === 'ayuda'"><circle cx="12" cy="12" r="9"/><path d="M9.5 9.2a2.6 2.6 0 0 1 5 .9c0 1.7-2.5 2.3-2.5 3.9"/><path d="M12 17h.01"/></template>
    <template v-else-if="nombre === 'carpeta-abierta'"><path d="M3 8V6a2 2 0 0 1 2-2h4l2 2h7a2 2 0 0 1 2 2v1"/><path d="M3 8h18l-2 10a2 2 0 0 1-2 1.6H5A2 2 0 0 1 3 18z"/></template>
    <template v-else-if="nombre === 'grafo'"><circle cx="6" cy="6" r="2.5"/><circle cx="18" cy="7" r="2.5"/><circle cx="12" cy="18" r="2.5"/><path d="M8.3 7.1 15.6 7M7.4 8.2l3.5 7.6M16.9 9.2l-3.6 6.7"/></template>
    <template v-else-if="nombre === 'lista'"><path d="M8 6h13M8 12h13M8 18h13"/><path d="M3.5 6h.01M3.5 12h.01M3.5 18h.01"/></template>
    <template v-else-if="nombre === 'casa'"><path d="M3 11 12 4l9 7"/><path d="M5 10v10h14V10"/></template>
    <template v-else-if="nombre === 'centrar'"><circle cx="12" cy="12" r="3"/><path d="M12 2v4M12 18v4M2 12h4M18 12h4"/></template>
    <template v-else-if="nombre === 'papelera'"><path d="M4 7h16"/><path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12"/><path d="M9 7V4h6v3"/><path d="M10 12l4 4M14 12l-4 4"/></template>
    <template v-else-if="nombre === 'reloj'"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></template>
    <template v-else-if="nombre === 'copias'"><rect x="8" y="8" width="12" height="13" rx="2"/><path d="M16 8V5a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h2"/></template>
    <template v-else-if="nombre === 'paleta'"><path d="M12 3a9 9 0 1 0 0 18c1.1 0 1.6-.8 1.6-1.6 0-.5-.2-.9-.5-1.2a1.6 1.6 0 0 1 1.2-2.7H16a5 5 0 0 0 5-5c0-4.1-4-7.5-9-7.5z"/><circle cx="7.5" cy="11" r="1"/><circle cx="10" cy="7" r="1"/><circle cx="15" cy="7.5" r="1"/></template>
    <template v-else-if="nombre === 'quitar'"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M9 13h6"/></template>
    <template v-else-if="nombre === 'guia'"><circle cx="12" cy="12" r="9"/><path d="m15.5 8.5-2 5-5 2 2-5z"/></template>
    <template v-else-if="nombre === 'terminal'"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/></template>
    <template v-else-if="nombre === 'archivo'"><path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4"/></template>
  </svg>
</template>
