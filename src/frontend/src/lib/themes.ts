import type { Theme } from "./types";

export const platformThemes: Record<string, Theme> = {
  snapchat: {
    name: "Snapchat",
    accent: "#fffc00", dark: "#cfc600", soft: "#fffdd9", ink: "#181712",
    bar: "bg-snapchat",
    badge: "bg-snapchat text-archive-ink border-snapchat-dark",
    card: "hover:border-snapchat-dark",
  },
  instagram: {
    name: "Instagram",
    accent: "#dd2a7b", dark: "#8134af", soft: "#fdeef5", ink: "#ffffff",
    bar: "bg-gradient-to-r from-[#f58529] via-instagram to-[#515bd4]",
    badge: "bg-instagram-soft text-instagram border-instagram",
    card: "hover:border-instagram",
  },
  facebook: {
    name: "Facebook",
    accent: "#1877f2", dark: "#0e5fcb", soft: "#e9f2fe", ink: "#ffffff",
    bar: "bg-facebook",
    badge: "bg-facebook-soft text-facebook border-facebook",
    card: "hover:border-facebook",
  },
};
