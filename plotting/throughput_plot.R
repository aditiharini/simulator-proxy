install.packages("ggplot2", repos="http://cran.us.r-project.org")
install.packages("dplyr", repos="http://cran.us.r-project.org")

library(ggplot2)
library(dplyr)

args = commandArgs(trailingOnly=TRUE)

data = read.csv(args[1])
png(args[2], width=960, height=480)
plot <- ggplot() +
  geom_point(data=data, aes(x=time, y=count), shape='.') +
  xlab("time (ms)") + 
  ylab("count")
print(plot)
dev.off()
