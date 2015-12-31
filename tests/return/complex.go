package main

// Complex types are not supported becuase they do no get marshalled into any
// JSON. The actual result comes out blank.

func Complex64() complex64 {
  return complex(1.2, 3.4)
}

// Since we cannot guarentee which order the functions will be processed in I
// am going to comment this out so that the error message will always contains
// complex64.

// func Complex128() complex128 {
//   return complex(5.6, 7.8)
// }
