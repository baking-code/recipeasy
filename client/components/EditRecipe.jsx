import React, { Component, PropTypes } from "react";
import _ from "lodash";
import { v4 as uuid } from "node-uuid";
import { connect } from "react-redux";
import Dropzone from "react-dropzone";

import { Card, Col, Row, Collection, CollectionItem, Icon, Button } from "react-materialize";
import Tags from "./Tags";
import EditList from "./EditList";
import { editActiveRecipeAction, editRecipeAction, toggleEditMode } from "../actions";


class EditRecipe extends Component {
  constructor(props) {
    super(props);
    this.state = {...props.recipe};
    this.editRecipeCollection = this.editRecipeCollection.bind(this);
    this.removeFromCollection = this.removeFromCollection.bind(this);
    this.addToCollection = this.addToCollection.bind(this);
  }


  editRecipe(recipe) {
    const { dispatch } = this.props;
    dispatch(editActiveRecipeAction(recipe));
  }

  editRecipeCollection(collectionName, index, value) {
    const recipe = {...this.props.recipe}
    recipe[collectionName][index] = value;

    this.editRecipe(recipe);
  }

  addToCollection(collectionName) {
    const recipe = {...this.props.recipe}
    recipe[collectionName].push("");

    this.editRecipe(recipe);
  }

  addImage(image) {
    getDataUri(image, (encoded) => {
      const recipe = {
        ...this.props.recipe,
        image: encoded
      };
      debugger;
      this.editRecipe(recipe);
    })
  }

  removeFromCollection(collectionName, index) {
    const recipe = {...this.props.recipe};
    const arr = recipe[collectionName];
    const modified = [...arr.slice(0, index), ...arr.slice(index + 1)];
    recipe[collectionName] = modified;
    this.editRecipe(recipe);
  }

  removeImage() {
    const recipe = {
      ...this.props.recipe,
      image: ""
    };
    this.editRecipe(recipe);
  }

  saveRecipe(recipe) {
    const { dispatch } = this.props;
    dispatch(editRecipeAction(recipe));
  }

  updateTags(tags) {
    const recipe = {...this.props.recipe};
    recipe.tags = tags;
    this.editRecipe(recipe);
  }

  render() {
    const { recipe, dispatch } = this.props;
    const imageArea = recipe.image ?
      (<div><img src={recipe.image} width={250} height={250}/><Button onClick={() => this.removeImage()} icon="cancel"/></div>) :
      (<Dropzone accept="image/*" onDrop={(f) => this.addImage(f)}><Icon>plus</Icon></Dropzone>);

    return (
      <div className="lime lighten-4">
        <Row>
          <Col s={6} offset="s2">
            <Card
              title={
                <input
                  defaultValue={recipe.name}
                  placeholder="Enter title"
                  onChange={(evt) => (this.editRecipe({...recipe, name: evt.target.value }))}
                />
              }
            >
              <input
                defaultValue={recipe.description}
                placeholder="Enter description"
                onChange={(evt) => (this.editRecipe({...recipe, description: evt.target.value }))}
              />
            </Card>
          </Col>
          <Col s={4} >
            {imageArea}
          </Col>
        </Row>
        <Row>
          <Col s={3} offset="s2">
            <EditList
              items={recipe.ingredients}
              title="Ingredients"
              editRecipeCollection={this.editRecipeCollection}
              removeFromCollection={this.removeFromCollection}
              addToCollection={this.addToCollection}
            />
          </Col>
          <Col s={5}>
            <EditList
              items={recipe.method}
              title="Method"
              editRecipeCollection={this.editRecipeCollection}
              removeFromCollection={this.removeFromCollection}
              addToCollection={this.addToCollection}
            />
          </Col>
        </Row>
        <Row><Col s={10} offset="s2">
          <Tags tags={recipe.tags} updateTags={(tags) => this.updateTags(tags)}/>
        </Col></Row>
        <Button
          floating
          icon="save"
          className="lime lighten-1"
          large
          style={{ bottom: "90px", right: "24px", position: "absolute" }}
          onClick={() => {this.saveRecipe(recipe); toggleEditMode(); }}
        />
        <Button
          floating
          icon="cancel"
          className="purple darken-1"
          large
          style={{ bottom: "25px", right: "24px", position: "absolute" }}
          onClick={() => dispatch(toggleEditMode())}
        />
      </div>
    );
  }
}

const getDataUri = (files, callback) => {
    if (files && files[0]) {
      var reader = new FileReader();
      reader.onload = function(e) {
           callback(e.target.result)
      };
      reader.onerror = function(e) {
           callback(null);
      };
      reader.readAsDataURL(files[0]);
  }
}

EditRecipe.propTypes = {
  recipe: PropTypes.object.isRequired
};

const wrap = connect();
export default wrap(EditRecipe);
