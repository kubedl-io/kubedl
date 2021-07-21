import path from "path";

function getModulePackageName(module) {
  if (!module.context) return null;
  const nodeModulesPath = path.join(__dirname, "../node_modules/");

  if (module.context.substring(0, nodeModulesPath.length) !== nodeModulesPath) {
    return null;
  }

  const moduleRelativePath = module.context.substring(nodeModulesPath.length);
  const [moduleDirName] = moduleRelativePath.split(path.sep);
  let packageName = moduleDirName; // handle tree shaking

  if (packageName && packageName.match("^_")) {
    packageName = packageName.match(/^_(@?[^@]+)/)[1];
  }

  return packageName;
}

export const webpackPlugin = (config) => {
  // optimize chunks
  config.optimization // share the same chunks across different modules
    .runtimeChunk(false)
    .splitChunks({
      chunks: "async",
      name: "vendors",
      maxInitialRequests: Infinity,
      minSize: 0,
      cacheGroups: {
        vendors: {
          test: (module) => {
            const packageName = getModulePackageName(module) || "";

            if (packageName) {
              return [
                "bizcharts",
                "gg-editor",
                "g6",
                "@antv",
                "gg-editor-core",
                "bizcharts-plugin-slider",
              ].includes(packageName);
            }

            return false;
          },

          name(module) {
            const packageName = getModulePackageName(module);

            if (packageName) {
              if (["bizcharts", "@antv_data-set"].indexOf(packageName) >= 0) {
                return "viz"; // visualization package
              }
            }

            return "misc";
          },
        },
      },
    });
};
