import { useRef, type ReactNode } from "react";
import { useHoloTilt } from "../hooks/useHoloTilt";

export function HoloProvider({ children }: { children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  useHoloTilt(ref);

  return (
    <div ref={ref} style={{ minHeight: "100%" }}>
      {children}
    </div>
  );
}
