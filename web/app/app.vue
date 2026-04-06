<template>
  <NuxtPage />
</template>

<style>
/* Global styles: dark theme, monospace font, full height */
html, body, #__nuxt {
  margin: 0;
  padding: 0;
  height: 100%;
  background: #0d1117;
  color: #c9d1d9;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', 'Menlo', monospace;
}

*, *::before, *::after {
  box-sizing: border-box;
}

/* Reset focus outline for terminal interaction */
:focus {
  outline: none;
}

/* Scrollbar styling */
::-webkit-scrollbar {
  width: 8px;
}

::-webkit-scrollbar-track {
  background: #0d1117;
}

::-webkit-scrollbar-thumb {
  background: #21262d;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #30363d;
}
</style>
