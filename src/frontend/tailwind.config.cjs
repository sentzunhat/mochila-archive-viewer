/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./index.html", "./src/**/*.{svelte,ts,js}"],
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
        snap:      "#ffef7b",
        "snap-dark": "#d6bf3c",
      },
      fontFamily: {
        serif: ['"Iowan Old Style"', '"Palatino Linotype"', '"Book Antiqua"', 'Palatino', 'serif'],
        sans:  ['"Avenir Next"', '"Segoe UI"', 'ui-sans-serif', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
