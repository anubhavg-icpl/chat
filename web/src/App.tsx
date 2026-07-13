import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { ChatRooms } from "./pages/ChatRooms";
import { Dashboard } from "./pages/Dashboard";
import { Directory } from "./pages/Directory";
import { InstantMessage } from "./pages/InstantMessage";
import { Sessions } from "./pages/Sessions";
import { Users } from "./pages/Users";
import { WebApiKeys } from "./pages/WebApiKeys";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="users" element={<Users />} />
          <Route path="sessions" element={<Sessions />} />
          <Route path="chat" element={<ChatRooms />} />
          <Route path="directory" element={<Directory />} />
          <Route path="webapi" element={<WebApiKeys />} />
          <Route path="im" element={<InstantMessage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
