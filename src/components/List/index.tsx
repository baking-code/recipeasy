import React, { useEffect, useState } from "react";
import firebase from "../../firebase";
import List from "./list";
import "./index.css";

function App() {
  const [recipes, setRecipes] = useState([]);

  useEffect(() => {
    (async () => {
      const data = await firebase.database().ref().once("value");
      setRecipes(Object.values(data.val()));
    })();
  }, []);
  const content = recipes.length ? <List items={recipes} /> : <p>Loading</p>;
  return (
    <div className="App">
      <header className="App-header">Recipeasy</header>
      <main>{content}</main>
    </div>
  );
}

export default App;
