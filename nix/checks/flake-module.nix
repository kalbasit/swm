{
  perSystem =
    {
      self',
      ...
    }:
    {
      checks = self'.packages // self'.devShells;
    };
}
