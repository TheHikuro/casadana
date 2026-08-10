/// <reference types="vite/client" />

// Image module types (*.png, *.jpg, *.jpeg, *.gif, *.webp, *.svg, *.avif, ...)
// come from vite/client above. Re-declaring them here made TypeScript merge two
// declarations of the same wildcard modules, each with its own default export,
// which failed the build with "TS2300: Duplicate identifier".
