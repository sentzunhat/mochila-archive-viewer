/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./index.html", "./src/**/*.{svelte,ts,js}"],
  // Preflight off: the explorer still relies on the handcrafted styles in
  // style.css; utilities are layered on top without resetting them.
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        archive: {
          bg:    "#fbfaf2",
          panel: "#fffef8",
          line:  "#ded7ad",
          muted: "#6f6b58",
          ink:   "#181712",
        },
        snapchat: {
          DEFAULT: "#fffc00",
          dark:    "#cfc600",
          soft:    "#fffdd9",
        },
        instagram: {
          DEFAULT: "#dd2a7b",
          dark:    "#8134af",
          soft:    "#fdeef5",
        },
        facebook: {
          DEFAULT: "#1877f2",
          dark:    "#0e5fcb",
          soft:    "#e9f2fe",
        },
      },
      fontFamily: {
        serif: ['"Iowan Old Style"', '"Palatino Linotype"', '"Book Antiqua"', 'Palatino', 'serif'],
        sans:  ['"Avenir Next"', '"Segoe UI"', 'ui-sans-serif', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
