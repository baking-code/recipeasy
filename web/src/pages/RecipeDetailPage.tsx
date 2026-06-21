import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import Layout from "@/components/Layout";
import TimerButton from "@/components/TimerButton";
import { api, Recipe } from "@/lib/api";
import styles from "./RecipeDetailPage.module.css";

export default function RecipeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [recipe, setRecipe] = useState<Recipe | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!id) return;
    api.recipes.get(id)
      .then(setRecipe)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [id]);

  const handleDelete = async () => {
    if (!recipe) return;
    if (!confirm(`Delete "${recipe.title}"?`)) return;
    await api.recipes.delete(recipe.id);
    navigate("/");
  };

  const totalMins = recipe
    ? (recipe.prep_time_mins ?? 0) + (recipe.cook_time_mins ?? 0)
    : 0;

  if (loading) return <Layout back="/"><div className={styles.loading}>Loading...</div></Layout>;
  if (error || !recipe) return <Layout back="/"><div className={styles.error}>{error || "Recipe not found"}</div></Layout>;

  return (
    <Layout
      back="/"
      title={recipe.title}
      actions={
        <button className={styles.editBtn} onClick={() => navigate(`/recipes/${recipe.id}/edit`)}>
          Edit
        </button>
      }
    >
      {recipe.image_path && (
        <img className={styles.heroImg} src={recipe.image_path} alt={recipe.title} />
      )}

      <h1 className={styles.title}>{recipe.title}</h1>

      {recipe.description && (
        <p className={styles.description}>{recipe.description}</p>
      )}

      <div className={styles.meta}>
        {recipe.servings && <span>🍽 {recipe.servings} servings</span>}
        {recipe.prep_time_mins && <span>🧑‍🍳 Prep {recipe.prep_time_mins} min</span>}
        {recipe.cook_time_mins && <span>🔥 Cook {recipe.cook_time_mins} min</span>}
        {totalMins > 0 && <span>⏱ Total {totalMins} min</span>}
      </div>

      {recipe.tags.length > 0 && (
        <div className={styles.tags}>
          {recipe.tags.map((t) => (
            <span key={t} className={styles.tag}>{t}</span>
          ))}
        </div>
      )}

      {recipe.ingredients.length > 0 && (
        <section className={styles.section}>
          <h2 className={styles.sectionTitle}>Ingredients</h2>
          <ul className={styles.ingredientList}>
            {recipe.ingredients.map((ing) => (
              <li key={ing.id ?? ing.position} className={styles.ingredient}>
                <span className={styles.ingredientAmount}>
                  {[ing.quantity, ing.unit].filter(Boolean).join(" ")}
                </span>
                <span>{ing.name}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {recipe.steps.length > 0 && (
        <section className={styles.section}>
          <h2 className={styles.sectionTitle}>Method</h2>
          <ol className={styles.stepList}>
            {recipe.steps.map((step) => (
              <li key={step.id ?? step.position} className={styles.step}>
                <span className={styles.stepNum}>{step.position}</span>
                <div className={styles.stepBody}>
                  <p>{step.instruction}</p>
                  {step.timer_minutes && (
                    <TimerButton
                      minutes={step.timer_minutes}
                      label={step.timer_label}
                    />
                  )}
                </div>
              </li>
            ))}
          </ol>
        </section>
      )}

      {recipe.source_url && (
        <p className={styles.source}>
          Source: <a href={recipe.source_url} target="_blank" rel="noopener noreferrer">{recipe.source_url}</a>
        </p>
      )}

      <button className={styles.deleteBtn} onClick={handleDelete}>
        Delete recipe
      </button>
    </Layout>
  );
}
