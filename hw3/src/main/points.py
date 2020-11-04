#!/usr/bin/python3
import turtle
import argparse
import json
import random

def main():
    parser = argparse.ArgumentParser(description="Manipulate point clouds")
    subparsers = parser.add_subparsers(help="sub-command",dest='cmd')

    generate_parser = subparsers.add_parser("generate",help="make a point cloud")
    generate_parser.add_argument("--output", help="filename",default="points.txt")
    generate_parser.add_argument("--clusters", help="number of clusters",default=7, type=int)
    generate_parser.add_argument("--minsize", help="minimum cluster size",default=500, type=int)
    generate_parser.add_argument("--maxsize", help="maximum cluster size",default=5000, type=int)

    display_parser = subparsers.add_parser("display",help="Display a points file")
    display_parser.add_argument("--input", help="filename",default="points.txt")

    display_parser = subparsers.add_parser("collect",help="Display a collection of JSON points files")
    display_parser.add_argument("--input", help="filename",nargs="+")

    args = parser.parse_args()

    cmds = {
        "generate": generate,
        "display": display,
        "collect": collect
    }
    cmds.get(args.cmd, lambda _: print("Please enter a subcommand"))(args)

def collect(args):
    points = []
    filenames = args.input
    for filename in filenames:
        with open(filename) as f:
            for line in f:
                data = json.loads(line)
                [x,y] = [float(v) for v in data["Value"].split()]
                cluster=int(data["Key"])
                points.append({"x":x,"y":y,"cluster":cluster})
    display_impl(points)

def generate(args):
    points = []
    for cluster in range(args.clusters):
        centerx = random.random()
        centery = random.random()
        for pointcount in range(random.randint(args.minsize, args.maxsize)):
            stddev = random.random() / 10
            pointx = random.gauss(centerx, stddev)
            pointy = random.gauss(centery, stddev)
            if 0<=pointx<1 and 0<=pointy<1:
                points.append({"x":pointx, "y":pointy, "cluster":0})
    with open(args.output,"w") as f:
        for point in points:
            f.write("%s %s %s\n" % (point["x"], point["y"], point["cluster"]))

def display(args):
    points = []
    with open(args.input) as f:
        for line in f:
            [x,y,cluster] = line.strip().split()
            points.append({"x":float(x),"y":float(y),"cluster":int(cluster)})
    display_impl(points)

def display_impl(points):
    colors = ["black","blue","red","green","yellow","purple","cyan","brown","tan"]
    turtle.tracer(0,0)
    turtle.hideturtle()
    for point in points:
        x = point["x"]
        y = point["y"]
        cluster = point["cluster"]
        scalex = turtle.window_width()
        scaley = turtle.window_height()
        turtle.color(colors[cluster % len(colors)])
        point = [x*scalex - scalex/2, y*scaley - scaley/2]
        turtle.pu()
        turtle.goto(point)
        turtle.pd()
        turtle.dot(2)
    turtle.update()
    turtle.exitonclick()

main()