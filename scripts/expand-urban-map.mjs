import { copyFileSync, readFileSync, writeFileSync } from 'node:fs';

const mapPath = 'frontend/src/assets/tiles/urban-kit/map.json';
const backupPath = 'frontend/src/assets/tiles/urban-kit/map.original.json';
const source = JSON.parse(readFileSync(mapPath, 'utf8'));

if (source.width !== 50 || source.height !== 50) {
  throw new Error(`Expected a 50x50 source map, got ${source.width}x${source.height}`);
}

copyFileSync(mapPath, backupPath);

const expanded = {
  ...source,
  width: 500,
  height: 500,
  layers: source.layers.map((layer) => ({
    ...layer,
    width: 500,
    height: 500,
    data: Array.from({ length: 500 * 500 }, (_, index) => {
      const row = Math.floor(index / 500);
      const column = index % 500;
      return layer.data[(row % 50) * 50 + (column % 50)];
    }),
  })),
};

writeFileSync(mapPath, `${JSON.stringify(expanded, null, 1)}\n`);
console.log(`Expanded ${mapPath} to ${expanded.width}x${expanded.height}`);
