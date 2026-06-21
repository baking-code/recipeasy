import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import Layout from "@/components/Layout";
import { api, Recipe } from "@/lib/api";
import styles from "./RecipeListPage.module.css";

export default function RecipeListPage() {
  const navigate = useNavigate();
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async (q: string) => {
    setLoading(true);
    setError("");
    try {
      const data = await api.recipes.list({ q });
      setRecipes(data);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load recipes");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => load(search), 300);
    return () => clearTimeout(timer);
  }, [search, load]);

  const totalMins = (r: Recipe) =>
    (r.prep_time_mins ?? 0) + (r.cook_time_mins ?? 0);

  return (
    <Layout
      actions={
        <div className={styles.headerActions}>
          <button
            className={styles.importBtn}
            onClick={() => navigate("/import")}
            title="Import recipe"
          >
            Import
          </button>
          <button
            className={styles.addBtn}
            onClick={() => navigate("/recipes/new")}
          >
            + New
          </button>
        </div>
      }
    >
      <div className={styles.searchWrap}>
        <input
          className={styles.search}
          type="search"
          placeholder="Search recipes..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          autoFocus
        />
      </div>

      {error && <p className={styles.error}>{error}</p>}

      {loading ? (
        <div className={styles.loading}>Loading...</div>
      ) : recipes.length === 0 ? (
        <div className={styles.empty}>
          {search ? "No recipes match your search." : "No recipes yet. Add your first one!"}
        </div>
      ) : (
        <ul className={styles.list}>
          {recipes.map((r) => (
            <li key={r.id}>
              <button
                className={styles.card}
                onClick={() => navigate(`/recipes/${r.id}`)}
              >
                {r.image_path && (
                  <img
                    className={styles.thumb}
                    src={r.image_path}
                    alt={r.title}
                    loading="lazy"
                  />
                )}
                <div className={styles.cardBody}>
                  <h2 className={styles.cardTitle}>{r.title}</h2>
                  {r.description && (
                    <p className={styles.cardDesc}>{r.description}</p>
                  )}
                  <div className={styles.cardMeta}>
                    {totalMins(r) > 0 && (
                      <span className={styles.tag}>⏱ {totalMins(r)} min</span>
                    )}
                    {r.tags.slice(0, 3).map((t) => (
                      <span key={t} className={styles.tag}>{t}</span>
                    ))}
                  </div>
                </div>
                <span className={styles.arrow}>›</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </Layout>
  );
}
