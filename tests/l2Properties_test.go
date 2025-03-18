package imptest_test

// Test the L2 properties:
// * a new imp will track all completion activity of a function under test
//   * all return values
//     * basic data types
//     * pointers
//     * slices
//     * structs
//     * nils
//     * nested versions of these
//     * arbitrary numbers of returns
//   * all panic values
//     * basic data types
//     * pointers
//     * slices
//     * structs
//     * nils
//     * nested versions of these
// * a new imp will track all call activity of any function from any dependency struct passed in
//   * the func name & args passed in
//     * basic data types
//     * pointers
//     * slices
//     * structs
//     * nils
//     * nested versions of these
// * a new imp will push any response to dependency calls
//   * return values
//   * panic values
// * a new imp will support checking and responding to arbitrarily concurrent calls
// * a new imp will fail a test cleanly if
//   * an expected function activity is not matched within the timeout
// * a new imp will panic if
//   * an expected function activity is incompatible with the function under test
//   * an expected function activity is incompatible with the expected dependency call's call signature
//   * a sent response is incompatible with the dependency call's return signature
