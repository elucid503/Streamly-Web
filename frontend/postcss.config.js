export default {

  plugins: {

    tailwindcss: {},
    "postcss-preset-env": {

      features: {

        "cascade-layers": true,
        "is-pseudo-class": true,
        "nesting-rules": true,

      },

    },
    autoprefixer: {},

  },

};
