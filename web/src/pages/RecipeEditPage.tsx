import { useEffect, useState, useRef } from "react";
import { useParams, useNavigate, useLocation } from "react-router-dom";
import Layout from "@/components/Layout";
import { api, Recipe, RecipeInput, Ingredient, Step } from "@/lib/api";
import styles from "./RecipeEditPage.module.css";

type IngredientDraft = Omit<Ingredient, "id">;
type StepDraft = Omit<Step, "id">;

function blankForm(): RecipeInput {
  return {
    title: "",
    description: "",
    servings: undefined,
    prep_time_mins: undefined,
    cook_time_mins: undefined,
    source_url: "",
    is_shared: true,
    tags: [],
    ingredients: [{ position: 1, name: "" }],
    steps: [{ position: 1, instruction: "" }],
  };
}

export default function RecipeEditPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const isNew = !id;

  const [form, setForm] = useState<RecipeInput>(
    // When coming from import page, the draft is passed via location state
    (location.state as { draft?: RecipeInput })?.draft ?? blankForm()
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string>("");
  const [tagInput, setTagInput] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!id) return;
    api.recipes.get(id).then((r: Recipe) => {
      setForm({
        title: r.title,
        description: r.description ?? "",
        servings: r.servings,
        prep_time_mins: r.prep_time_mins,
        cook_time_mins: r.cook_time_mins,
        source_url: r.source_url ?? "",
        is_shared: r.is_shared,
        tags: r.tags,
        ingredients: r.ingredients.map((i) => ({ position: i.position, quantity: i.quantity, unit: i.unit, name: i.name })),
        steps: r.steps.map((s) => ({ position: s.position, instruction: s.instruction, timer_minutes: s.timer_minutes, timer_label: s.timer_label })),
      });
      if (r.image_path) setImagePreview(r.image_path);
    });
  }, [id]);

  const set = <K extends keyof RecipeInput>(key: K, val: RecipeInput[K]) =>
    setForm((f) => ({ ...f, [key]: val }));

  const updateIngredient = (i: number, patch: Partial<IngredientDraft>) =>
    setForm((f) => {
      const ingredients = [...f.ingredients];
      ingredients[i] = { ...ingredients[i], ...patch };
      return { ...f, ingredients };
    });

  const addIngredient = () =>
    setForm((f) => ({
      ...f,
      ingredients: [...f.ingredients, { position: f.ingredients.length + 1, name: "" }],
    }));

  const removeIngredient = (i: number) =>
    setForm((f) => ({
      ...f,
      ingredients: f.ingredients.filter((_, idx) => idx !== i).map((ing, idx) => ({ ...ing, position: idx + 1 })),
    }));

  const updateStep = (i: number, patch: Partial<StepDraft>) =>
    setForm((f) => {
      const steps = [...f.steps];
      steps[i] = { ...steps[i], ...patch };
      return { ...f, steps };
    });

  const addStep = () =>
    setForm((f) => ({
      ...f,
      steps: [...f.steps, { position: f.steps.length + 1, instruction: "" }],
    }));

  const removeStep = (i: number) =>
    setForm((f) => ({
      ...f,
      steps: f.steps.filter((_, idx) => idx !== i).map((s, idx) => ({ ...s, position: idx + 1 })),
    }));

  const addTag = () => {
    const tag = tagInput.trim().toLowerCase();
    if (tag && !form.tags.includes(tag)) {
      set("tags", [...form.tags, tag]);
    }
    setTagInput("");
  };

  const removeTag = (tag: string) =>
    set("tags", form.tags.filter((t) => t !== tag));

  const handleImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setImageFile(file);
    setImagePreview(URL.createObjectURL(file));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.title.trim()) {
      setError("Title is required");
      return;
    }
    setSaving(true);
    setError("");
    try {
      const recipe = isNew
        ? await api.recipes.create(form)
        : await api.recipes.update(id!, form);

      if (imageFile) {
        await api.recipes.uploadImage(recipe.id, imageFile);
      }

      navigate(`/recipes/${recipe.id}`);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to save recipe");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Layout back={id ? `/recipes/${id}` : "/"} title={isNew ? "New Recipe" : "Edit Recipe"}>
      <form className={styles.form} onSubmit={handleSubmit}>
        {error && <p className={styles.error}>{error}</p>}

        {/* Title */}
        <div className={styles.field}>
          <label className={styles.label}>Title *</label>
          <input
            className={styles.input}
            value={form.title}
            onChange={(e) => set("title", e.target.value)}
            placeholder="e.g. Spaghetti Bolognese"
            required
          />
        </div>

        {/* Description */}
        <div className={styles.field}>
          <label className={styles.label}>Description</label>
          <textarea
            className={styles.textarea}
            value={form.description ?? ""}
            onChange={(e) => set("description", e.target.value)}
            placeholder="A short description..."
            rows={2}
          />
        </div>

        {/* Timings */}
        <div className={styles.row}>
          <div className={styles.field}>
            <label className={styles.label}>Prep time (min)</label>
            <input
              className={styles.input}
              type="number"
              min={0}
              value={form.prep_time_mins ?? ""}
              onChange={(e) => set("prep_time_mins", e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
          <div className={styles.field}>
            <label className={styles.label}>Cook time (min)</label>
            <input
              className={styles.input}
              type="number"
              min={0}
              value={form.cook_time_mins ?? ""}
              onChange={(e) => set("cook_time_mins", e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
          <div className={styles.field}>
            <label className={styles.label}>Servings</label>
            <input
              className={styles.input}
              type="number"
              min={1}
              value={form.servings ?? ""}
              onChange={(e) => set("servings", e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
        </div>

        {/* Image */}
        <div className={styles.field}>
          <label className={styles.label}>Photo</label>
          {imagePreview && (
            <img className={styles.imagePreview} src={imagePreview} alt="Recipe" />
          )}
          <button type="button" className={styles.uploadBtn} onClick={() => fileRef.current?.click()}>
            {imagePreview ? "Change photo" : "Add photo"}
          </button>
          <input
            ref={fileRef}
            type="file"
            accept="image/*"
            style={{ display: "none" }}
            onChange={handleImageChange}
          />
        </div>

        {/* Tags */}
        <div className={styles.field}>
          <label className={styles.label}>Tags</label>
          <div className={styles.tagRow}>
            {form.tags.map((t) => (
              <span key={t} className={styles.tagChip}>
                {t}
                <button type="button" className={styles.tagRemove} onClick={() => removeTag(t)}>×</button>
              </span>
            ))}
            <input
              className={styles.tagInput}
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === ",") { e.preventDefault(); addTag(); } }}
              placeholder="Add tag..."
            />
          </div>
        </div>

        {/* Ingredients */}
        <div className={styles.field}>
          <label className={styles.label}>Ingredients</label>
          {form.ingredients.map((ing, i) => (
            <div key={i} className={styles.ingredientRow}>
              <input
                className={`${styles.input} ${styles.qty}`}
                value={ing.quantity ?? ""}
                onChange={(e) => updateIngredient(i, { quantity: e.target.value || undefined })}
                placeholder="Qty"
              />
              <input
                className={`${styles.input} ${styles.unit}`}
                value={ing.unit ?? ""}
                onChange={(e) => updateIngredient(i, { unit: e.target.value || undefined })}
                placeholder="Unit"
              />
              <input
                className={`${styles.input} ${styles.ingName}`}
                value={ing.name}
                onChange={(e) => updateIngredient(i, { name: e.target.value })}
                placeholder="Ingredient"
                required
              />
              <button
                type="button"
                className={styles.removeBtn}
                onClick={() => removeIngredient(i)}
                disabled={form.ingredients.length <= 1}
              >
                ×
              </button>
            </div>
          ))}
          <button type="button" className={styles.addRowBtn} onClick={addIngredient}>
            + Add ingredient
          </button>
        </div>

        {/* Steps */}
        <div className={styles.field}>
          <label className={styles.label}>Method</label>
          {form.steps.map((step, i) => (
            <div key={i} className={styles.stepRow}>
              <span className={styles.stepNum}>{i + 1}</span>
              <div className={styles.stepFields}>
                <textarea
                  className={`${styles.textarea} ${styles.stepText}`}
                  value={step.instruction}
                  onChange={(e) => updateStep(i, { instruction: e.target.value })}
                  placeholder={`Step ${i + 1}`}
                  rows={2}
                  required
                />
                <div className={styles.timerRow}>
                  <input
                    className={`${styles.input} ${styles.timerInput}`}
                    type="number"
                    min={0}
                    value={step.timer_minutes ?? ""}
                    onChange={(e) => updateStep(i, { timer_minutes: e.target.value ? Number(e.target.value) : undefined })}
                    placeholder="Timer (min)"
                  />
                  {step.timer_minutes && (
                    <input
                      className={`${styles.input} ${styles.timerLabel}`}
                      value={step.timer_label ?? ""}
                      onChange={(e) => updateStep(i, { timer_label: e.target.value || undefined })}
                      placeholder="Timer label"
                    />
                  )}
                </div>
              </div>
              <button
                type="button"
                className={styles.removeBtn}
                onClick={() => removeStep(i)}
                disabled={form.steps.length <= 1}
              >
                ×
              </button>
            </div>
          ))}
          <button type="button" className={styles.addRowBtn} onClick={addStep}>
            + Add step
          </button>
        </div>

        {/* Shared toggle */}
        <label className={styles.checkRow}>
          <input
            type="checkbox"
            checked={form.is_shared}
            onChange={(e) => set("is_shared", e.target.checked)}
          />
          Share with family
        </label>

        <button type="submit" className={styles.saveBtn} disabled={saving}>
          {saving ? "Saving..." : isNew ? "Create Recipe" : "Save Changes"}
        </button>
      </form>
    </Layout>
  );
}
