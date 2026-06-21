import { Link, useNavigate } from "react-router-dom";
import { clearToken, getUser } from "@/lib/api";
import styles from "./Layout.module.css";

interface LayoutProps {
  children: React.ReactNode;
  title?: string;
  back?: string;
  actions?: React.ReactNode;
}

export default function Layout({ children, title, back, actions }: LayoutProps) {
  const navigate = useNavigate();
  const user = getUser();

  const handleLogout = () => {
    clearToken();
    navigate("/login", { replace: true });
  };

  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <div className={styles.headerLeft}>
          {back ? (
            <button className={styles.backBtn} onClick={() => navigate(back)}>
              ← Back
            </button>
          ) : (
            <Link to="/" className={styles.brand}>Recipeasy</Link>
          )}
        </div>
        {title && <h1 className={styles.headerTitle}>{title}</h1>}
        <div className={styles.headerRight}>
          {actions}
          {!back && (
            <button className={styles.logoutBtn} onClick={handleLogout} title={user?.email}>
              {user?.name?.split(" ")[0]}
            </button>
          )}
        </div>
      </header>
      <main className={styles.main}>{children}</main>
    </div>
  );
}
