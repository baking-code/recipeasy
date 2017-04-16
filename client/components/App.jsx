import React, { Component } from "react";
import { Link } from "react-router";
import styled from "styled-components";
import { Row } from "react-flexbox-grid";

import "node_modules/roboto-npm-webfont/style.scss";
import { primary } from "./constants/colours";


class App extends Component {

  render() {
    return (
      <div>
        <Nav>
        <Row center="xs">
          <Title to={"/recipes"}>Recipes</Title>
          </Row>
        </Nav>
        <main>
          {this.props.children}
        </main>
      </div>
    );
  }
}

const Nav = styled.nav`
  box-shadow: 0 2px 5px 0 rgba(0,0,0,0.16),0 2px 10px 0 rgba(0,0,0,0.12);
  display: block;
  height: 64px;
  line-height: 64px;
  background-color: ${primary};
  margin-bottom: 20px
`;

const Title = styled(Link)`
  font-size: xx-large;
  color: #FFF;
  text-decoration: none;
`;

export default App;