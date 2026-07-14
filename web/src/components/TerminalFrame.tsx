import type { ReactNode } from "react";

export function TerminalFrame({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="terminal">
      <div className="terminal-bar">
        <span className="dots">
          <i />
          <i />
          <i />
        </span>
        <span className="terminal-title">{title}</span>
      </div>
      <div className="terminal-body">{children}</div>
    </section>
  );
}
