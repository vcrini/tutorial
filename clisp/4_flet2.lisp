(print 
  (flet ((f (n)
	  (+ n 10))
	  (g (n)
	     (- n 3)))
    (g (f 5)))
       )
