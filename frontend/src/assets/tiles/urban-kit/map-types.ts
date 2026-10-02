export type TileLayer = {
  name: string;
  data: number[];
};

export type TileMap = {
  layers: TileLayer[];
};
