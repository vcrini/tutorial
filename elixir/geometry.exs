defmodule Rectangle do
  def area({h, w}), do: h*w
  def perimeter({h, w}) do
    2*(h+w)
  end
defmodule Square do 
  def area({w}), do: Rectangle.area({w, w})
  def area({w, h}) when w == h do
    Rectangle.area({w, w})
  end
  def perimeter({w}) do
    Rectangle.perimeter({w, w})
  end
  def perimeter({w, h}) when w == h do 
    Rectangle.perimeter({w, w})
  end
end

r= {3, 4}
IO.puts "L'area del rettangolo #{inspect r} è #{Rectangle.area r}"
