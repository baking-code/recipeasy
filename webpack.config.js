
/* eslint-disable no-var, vars-on-top */
var webpack = require("webpack");
var path = require("path");
global.Promise = require("bluebird");

var autoprefixer = require("autoprefixer");
var ExtractTextPlugin = require("extract-text-webpack-plugin");

var sassLoaders = [
  "css-loader?sourceMap",
  "postcss-loader",
  "sass-loader?sourceMap&includePaths[]=" + path.resolve(__dirname, "./client")
];

module.exports = {
  context: path.join(__dirname, "./client"),
  entry: {
    jsx: "./index.js",
    html: "./index.html",
    vendor: [
      "react",
      "react-dom",
      "redux",
      "react-redux",
      "react-router",
      "react-router-redux"
    ]
  },
  output: {
    path: path.join(__dirname, "./dist"),
    filename: "bundle.js"
  },
  module: {
    loaders: [
      {
        test: /\.html$/,
        loader: "file?name=[name].[ext]"
      },
      {
        test: /\.(js|jsx)$/,
        exclude: /node_modules/,
        loader: 'babel',
        query: { presets: [ 'es2015', 'react' ] }
      },
      {
        test: /\.scss$/,
        loader: ExtractTextPlugin.extract("style-loader", sassLoaders.join("!"))
      }
    ]
  },
  resolve: {
    extensions: ["", ".js", ".jsx"],
    root: [path.resolve(__dirname), path.resolve(__dirname, "node_modules")]
  },
  plugins: [
    new webpack.optimize.CommonsChunkPlugin("vendor", "vendor.bundle.js"),
    new webpack.DefinePlugin({
      "process.env": { NODE_ENV: JSON.stringify(process.env.NODE_ENV || "development") }
    }),
    new ExtractTextPlugin("style.css")
  ],
  postcss: [
    autoprefixer({
      browsers: ["last 2 versions"]
    })
  ],
  devServer: {
    contentBase: "./client"
  }
}
