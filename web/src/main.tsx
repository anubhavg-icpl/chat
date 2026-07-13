import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { HoloProvider } from "./components/HoloProvider";
import { useCursorSparkles } from "./hooks/useCursorSparkles";
import "./styles/global.css";

function Root() {
  useCursorSparkles();
  return (
    <HoloProvider>
      <App />
    </HoloProvider>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
