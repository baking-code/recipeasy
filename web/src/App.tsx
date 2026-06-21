import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { isAuthenticated } from "@/lib/api";
import LoginPage from "@/pages/LoginPage";
import RecipeListPage from "@/pages/RecipeListPage";
import RecipeDetailPage from "@/pages/RecipeDetailPage";
import RecipeEditPage from "@/pages/RecipeEditPage";
import ImportPage from "@/pages/ImportPage";

function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!isAuthenticated()) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <RecipeListPage />
            </RequireAuth>
          }
        />
        <Route
          path="/recipes/new"
          element={
            <RequireAuth>
              <RecipeEditPage />
            </RequireAuth>
          }
        />
        <Route
          path="/recipes/:id"
          element={
            <RequireAuth>
              <RecipeDetailPage />
            </RequireAuth>
          }
        />
        <Route
          path="/recipes/:id/edit"
          element={
            <RequireAuth>
              <RecipeEditPage />
            </RequireAuth>
          }
        />
        <Route
          path="/import"
          element={
            <RequireAuth>
              <ImportPage />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
