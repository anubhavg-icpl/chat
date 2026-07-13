import { useEffect, type RefObject } from "react";

/** crow.rip-style holographic tilt + pointer tracking for neon foil cards */
export function useHoloTilt(rootRef: RefObject<HTMLElement | null>) {
  useEffect(() => {
    const root = rootRef.current ?? document;
    const cards = () =>
      Array.from(
        (root instanceof Document ? root : root).querySelectorAll<HTMLElement>(
          ".holo-card",
        ),
      );

    const cleanups: Array<() => void> = [];

    function bind(card: HTMLElement) {
      if (card.dataset.holoBound === "1") return;
      card.dataset.holoBound = "1";

      let hovering = false;

      const apply = (x: number, y: number, rect: DOMRect) => {
        const mx = x / rect.width;
        const my = y / rect.height;
        card.style.setProperty("--mx", mx.toFixed(3));
        card.style.setProperty("--my", my.toFixed(3));
        card.style.setProperty("--pointer-x", `${(mx * 100).toFixed(1)}%`);
        card.style.setProperty("--pointer-y", `${(my * 100).toFixed(1)}%`);
        const rotX = ((y - rect.height / 2) / (rect.height / 2)) * 8;
        const rotY = ((x - rect.width / 2) / (rect.width / 2)) * 12;
        card.style.transform = `perspective(1200px) translateY(-6px) rotateX(${-rotX}deg) rotateY(${rotY}deg) scale(1.02)`;
      };

      const reset = () => {
        hovering = false;
        card.style.setProperty("--holo-intensity", "0");
        card.style.setProperty("--mx", "0.5");
        card.style.setProperty("--my", "0.5");
        card.style.setProperty("--pointer-x", "50%");
        card.style.setProperty("--pointer-y", "50%");
        card.style.transform =
          "perspective(1200px) translateY(0) rotateX(0) rotateY(0) scale(1)";
      };

      const onEnter = () => {
        hovering = true;
        card.style.setProperty("--holo-intensity", "1");
      };

      const onMove = (e: MouseEvent) => {
        if (!hovering) return;
        const rect = card.getBoundingClientRect();
        apply(e.clientX - rect.left, e.clientY - rect.top, rect);
      };

      card.addEventListener("mouseenter", onEnter);
      card.addEventListener("mousemove", onMove);
      card.addEventListener("mouseleave", reset);

      cleanups.push(() => {
        card.removeEventListener("mouseenter", onEnter);
        card.removeEventListener("mousemove", onMove);
        card.removeEventListener("mouseleave", reset);
        delete card.dataset.holoBound;
      });
    }

    const observer = new MutationObserver(() => {
      cards().forEach(bind);
    });

    cards().forEach(bind);
    observer.observe(document.body, { childList: true, subtree: true });

    return () => {
      observer.disconnect();
      cleanups.forEach((fn) => fn());
    };
  }, [rootRef]);
}
