/* global module require __dirname */

const path = require('path')
const webpack = require('webpack')
const CopyPlugin = require('copy-webpack-plugin')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const JsonMinimizerPlugin = require('json-minimizer-webpack-plugin')
const TerserPlugin = require('terser-webpack-plugin')
const {VueLoaderPlugin} = require('vue-loader')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')

const srcDir = path.join(__dirname, '..', 'src')
const publicDir = path.join(__dirname, '..', 'public')
const assetsDir = path.join(publicDir, 'assets')
const distDir = path.join(__dirname, '..', 'dist')

module.exports = {
  node: {
    global: false,
  },
  entry: {
    index: path.join(srcDir, 'index.ts'),
  },
  output: {
    path: distDir,
    filename: 'scripts.js?v=[chunkhash]',
    clean: true,
  },
  optimization: {
    minimize: true,
    splitChunks: {
      name: 'vendor',
    },
    minimizer: [
      new TerserPlugin({
        extractComments: false,
      }),
      new JsonMinimizerPlugin({
        test: /\.json$/i,
      }),
    ]
  },
  resolve: {
    extensions: ['.ts', '.js'],
  },
  module: {
    rules: [
      {
        test: /\.ts$/,
        use: {
          loader: 'ts-loader',
          options: {
            appendTsSuffixTo: [/\.vue$/],
          },
        },
        exclude: /node_modules/,
      },
      {
        test: /\.css$/,
        use: [MiniCssExtractPlugin.loader, 'css-loader'],
      },
      {
        test: /\.scss$/,
        use: [MiniCssExtractPlugin.loader, 'css-loader', 'sass-loader'],
      },
      {
        test: /\.vue$/,
        loader: 'vue-loader'
      },
    ],
  },
  plugins: [
    new webpack.IgnorePlugin({resourceRegExp: /^\.\/locale$/, contextRegExp: /moment$/}), // https://github.com/jmblog/how-to-optimize-momentjs-with-webpack
    new webpack.DefinePlugin({ // https://github.com/vuejs/vue-next/tree/master/packages/vue#bundler-build-feature-flags
      __VUE_OPTIONS_API__: true,
      __VUE_PROD_DEVTOOLS__: false,
    }),
    new VueLoaderPlugin(),
    new MiniCssExtractPlugin({ // https://github.com/webpack-contrib/mini-css-extract-plugin
      filename: 'styles.css?v=[contenthash]'
    }),
    new HtmlWebpackPlugin({ // https://github.com/jantimon/html-webpack-plugin#options
      inject: 'body',
      chunks: ['index'],
      template: path.join(publicDir, 'index.html'),
      minify: {
        minifyCSS: true,
        collapseWhitespace: true,
        keepClosingSlash: true,
        removeComments: true,
      }
    }),
    new CopyPlugin({
      patterns: [{from: '.', to: '.', context: assetsDir, globOptions: {ignore: ['**/*.md']}}],
    }),
  ],
}
