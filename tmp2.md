For the Taxonomy table:

* Drop the "what can be wrapped" moniker - it's not aaccuarte for most of it. Maybe replace it with "Function
  Interactions". 
* For the types, we can drop Function Type and Struct.
* We should think of Function vs Function Type and Interface vs Struct as another column. We can create wrappers that
  accept function types, and wrappers that accept interface types. We can extrapolate the function types from named
  (function) types, from function definitions, and from function literals. We can extrapolate the interface types from named (interface) types, from struct definitions with methods, and from struct literals with methods.
* the second column should be titled "As Target (Wrap)"
* the function row should have "TargetFunction" in the second column, and "DependencyFunction" in the third.
* the interface row should have "TargetInterface" in the second column, and "DepenencyInterface" in the third.
* the shared state row should say "actions on shared state are not interceptable" in all the columns >=2.
* the channel row... we could actually do channels. let's try it. Channel row 2 should be "TargetChannel", row 3 should
  be "DependencyChannel". We could defer implementation till after the major refactor, though.

For the Direction determins role table:

* Direction column should be "arg direction" and say "we set" for target and "we get" for dependency. 
* Add another column
  for "result direction" and say "we get" for target and "we set" for dependency.
