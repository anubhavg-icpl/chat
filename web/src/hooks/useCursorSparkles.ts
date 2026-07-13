import { useEffect } from "react";

/** light cursor spark trail inspired by crow.rip retro-fx */
export function useCursorSparkles() {
  useEffect(() => {
    if (!window.matchMedia("(hover: hover) and (pointer: fine)").matches) {
      return;
    }

    const hues = [45, 38, 30, 330, 280, 200];
    let last = 0;
    let lx = 0;
    let ly = 0;

    const onMove = (e: MouseEvent) => {
      const now = performance.now();
      const dx = e.clientX - lx;
      const dy = e.clientY - ly;
      const moved = Math.hypot(dx, dy);
      lx = e.clientX;
      ly = e.clientY;
      if (moved < 8 || now - last < 42) return;
      last = now;

      const s = document.createElement("span");
      s.className = "cursor-spark";
      s.style.left = `${e.clientX + (Math.random() - 0.5) * 12}px`;
      s.style.top = `${e.clientY + (Math.random() - 0.5) * 12}px`;
      s.style.setProperty("--spark-size", `${6 + Math.random() * 7}px`);
      s.style.setProperty(
        "--spark-hue",
        String(hues[(Math.random() * hues.length) | 0]),
      );
      s.style.setProperty("--spark-rot", `${Math.random() * 360}deg`);
      s.style.setProperty("--spark-drift", `${10 + Math.random() * 16}px`);
      s.style.setProperty("--spark-life", `${480 + Math.random() * 280}ms`);
      document.body.appendChild(s);
      s.addEventListener("animationend", () => s.remove(), { once: true });
    };

    document.addEventListener("mousemove", onMove, { passive: true });
    return () => document.removeEventListener("mousemove", onMove);
  }, []);
}
