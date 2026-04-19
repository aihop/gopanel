module.exports = {
    env: {
      node: true,
    },
    extends: ["eslint:recommended", "plugin:vue/vue3-recommended"],
    rules: {
      "comma-dangle": ["error", "always-multiline"],
      "vue/require-default-prop": false, // 关闭默认值提示
    },
  };
  