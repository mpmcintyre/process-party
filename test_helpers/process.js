(async () => {
  while (true) {
    console.log("JS running");
    await new Promise((r) => setTimeout(r, 3000));
  }
})();
