(print 
  (labels ((a (n)
	      (+ n 5))
	   (b (n)
	      (+ (a n) 6)))
    (b 10))
  )
