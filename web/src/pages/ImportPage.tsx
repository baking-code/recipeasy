import { useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import Layout from "@/components/Layout";
import { api, RecipeInput } from "@/lib/api";
import styles from "./ImportPage.module.css";

type ImportMode = "url" | "photo";

export default function ImportPage() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<ImportMode>("url");
  const [url, setUrl] = useState("");
  const [photoFile, setPhotoFile] = useState<File | null>(null);
  const [photoPreview, setPhotoPreview] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const handlePhotoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setPhotoFile(file);
    setPhotoPreview(URL.createObjectURL(file));
  };

  const handleImport = async () => {
    setError("");
    setLoading(true);
    try {
      let draft: RecipeInput;
      if (mode === "url") {
        if (!url.trim()) { setError("Enter a URL"); return; }
        draft = await api.import.fromURL(url.trim());
      } else {
        if (!photoFile) { setError("Choose a photo"); return; }
        draft = await api.import.fromPhoto(photoFile);
      }
      // Navigate to edit page with draft pre-filled for review before saving
      navigate("/recipes/new", { state: { draft } });
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Import failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Layout back="/" title="Import Recipe">
      <p className={styles.intro}>
        Import a recipe from a web URL or a photo of a written recipe. You'll review and edit it before saving.
      </p>

      <div className={styles.tabs}>
        <button
          className={`${styles.tab} ${mode === "url" ? styles.tabActive : ""}`}
          onClick={() => setMode("url")}
        >
          Web URL
        </button>
        <button
          className={`${styles.tab} ${mode === "photo" ? styles.tabActive : ""}`}
          onClick={() => setMode("photo")}
        >
          Photo
        </button>
      </div>

      {mode === "url" ? (
        <div className={styles.section}>
          <input
            className={styles.input}
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://www.bbcgoodfood.com/recipes/..."
            onKeyDown={(e) => e.key === "Enter" && handleImport()}
          />
          <p className={styles.hint}>
            Works with BBC Good Food, AllRecipes, NYT Cooking, Serious Eats, and most other recipe sites.
          </p>
        </div>
      ) : (
        <div className={styles.section}>
          {photoPreview ? (
            <img className={styles.preview} src={photoPreview} alt="Recipe photo" />
          ) : (
            <button className={styles.photoBtn} onClick={() => fileRef.current?.click()}>
              <span className={styles.photoIcon}>📷</span>
              <span>Tap to choose a photo</span>
              <span className={styles.photoHint}>Handwritten or printed recipe</span>
            </button>
          )}
          {photoPreview && (
            <button className={styles.changePhoto} onClick={() => fileRef.current?.click()}>
              Change photo
            </button>
          )}
          <input
            ref={fileRef}
            type="file"
            accept="image/*"
            capture="environment"
            style={{ display: "none" }}
            onChange={handlePhotoChange}
          />
        </div>
      )}

      {error && <p className={styles.error}>{error}</p>}

      <button
        className={styles.importBtn}
        onClick={handleImport}
        disabled={loading}
      >
        {loading ? (
          <span>Extracting recipe<span className={styles.dots}>...</span></span>
        ) : (
          "Extract Recipe"
        )}
      </button>
    </Layout>
  );
}
