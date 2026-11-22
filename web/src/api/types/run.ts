export type RunCreateRequest = {
  // The size of the screen of the client in device pixels, which bounds the screen versions of the captures
  screenWidth: number;
  screenHeight: number;
};

export type RunResponse = {
  run: number;
  imagesPerRun: number;
};
