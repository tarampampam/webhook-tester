/* global module require */

const { merge } = require('webpack-merge')
const common = require('./webpack.common.js')

module.exports = merge(common, {
  devtool: 'inline-source-map',
  mode: 'development',
  optimization: {
    minimize: false,
  },
  devServer: {
    proxy:
      {
        "/api": {
          target: 'http://web:8080',
        },
        "/ws/session": {
          target: 'http://web:8080',
          ws: true,
        },
      },
    port: 8081,
  },
})
