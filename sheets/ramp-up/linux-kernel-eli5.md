# Linux Kernel — ELI5

> The kernel is the boss of your computer that helps every program share the same toys without fighting.

## Prerequisites

(none — start here)

This sheet is the very first stop. You do not need to know anything about computers to read it. You do not need to know what a "command line" is. You do not need to know what "Linux" is. By the end of this sheet you will know all of those things in plain English, and you will have typed real commands into a real terminal and watched real things happen.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## Plain English

### Imagine your computer is a giant toy factory

Picture a really big factory. Inside the factory there are lots of workers. Each worker has a job. One worker makes red toys. Another worker makes blue toys. Another worker paints them. Another worker boxes them up. Another worker puts the boxes on a truck.

All these workers want to use the same things. They all want to use the same paint. They all want to use the same boxes. They all want to use the same trucks. If every worker just walked up to the paint shelf at the same time and grabbed paint, there would be paint everywhere. Workers would bump into each other. Some workers would never get paint. Other workers would steal paint that wasn't theirs.

So the factory has a manager. The manager stands at the front and says, "Worker #1, you get the red paint for ten seconds. Worker #2, wait your turn. Worker #3, you can have a box now. Worker #4, no, that box belongs to Worker #1, leave it alone."

The manager doesn't make any toys. The manager doesn't paint anything. The manager doesn't drive any trucks. The manager just makes sure all the workers can do their jobs without crashing into each other.

**That manager is the kernel.**

The workers are the programs on your computer. Your web browser is a worker. Your music player is a worker. Your text editor is a worker. The little clock in the corner of your screen is a worker. Every single one of them is a program, and every single program needs the kernel to manage them so they can all share the computer at the same time.

### Imagine your computer is a restaurant

Here is another way to think about it. Pretend your computer is a restaurant.

You sit down at a table. You are a program. You are hungry. You want food.

You do not get up and walk into the kitchen and start cooking. That would be a disaster. The kitchen has hot stoves. The kitchen has sharp knives. The kitchen has raw chicken. If you walked in there, you might burn yourself, or cut yourself, or get sick. Also, if every customer walked into the kitchen at the same time, there would be no room to cook anything. Customers would knock pots over. Customers would grab each other's food. The whole place would be a mess.

Instead, you raise your hand. A waiter comes to your table. You tell the waiter what you want. "I want a cheese pizza, please." The waiter writes it down, walks to the kitchen, tells the cook, waits for the food, and brings it back to your table. You never go in the kitchen. You never see the kitchen. You don't need to. The waiter handles everything for you.

**That waiter is the kernel.**

The kitchen is the hardware. The hardware is the actual physical parts of your computer: the CPU (the brain), the RAM (the short-term memory), the hard drive (the long-term memory), the screen, the keyboard, the speakers, the Wi-Fi card. All those parts are dangerous if a program touches them directly. So the kernel stands between every program and the hardware. Every time a program needs something from the hardware, the kernel goes and gets it.

### What if there were no kernel?

Without a kernel, every program would try to talk directly to the screen. Every program would try to grab keyboard keys before the other programs could see them. Every program would try to write files to the hard drive at the same place at the same time. Every program would think it was alone on the computer and would assume nothing else was running.

It would be like fifty kids trying to drive the same car at the same time. One pulls the wheel left. One pulls it right. One stomps on the gas. One stomps on the brake. One honks the horn. One opens the door while it's moving. The car crashes immediately.

The kernel is the grown-up driver. Only one driver per car. The kernel takes turns letting each kid say where they want to go, and the kernel does the actual driving. Nobody crashes.

### A different picture: the kernel as a librarian

Imagine a giant library, the biggest library you have ever seen. The library has millions of books. Every day, hundreds of people come in and want to borrow books, return books, look up books, and ask questions about books.

If every visitor walked into the storage rooms in the back and started rummaging around, the library would fall apart in an hour. Books would get lost. Books would get mis-shelved. Two people would grab the same book and tear it in half fighting over it. Visitors would knock over carts. Some shelves would be picked clean while others sat full forever.

So the library has librarians. The librarians stand at the desk. Visitors walk up and ask, "Can I have the book about dinosaurs?" The librarian goes back to the shelves, finds the book, brings it to the visitor, writes down who borrowed it, and tells them when to bring it back. Visitors never go in the back.

That is what the kernel does. The visitors are programs. The books are everything inside the computer (memory, files, network connections, the screen, the keyboard). The kernel is the librarian behind the desk. Every program asks. The kernel goes and fetches.

### A fourth picture: the kernel as a traffic cop

Imagine a really busy intersection in a city. Cars are coming from four directions. Pedestrians want to cross. Bicycles want to weave through. Trucks need to make wide turns. If everybody just went whenever they felt like it, there would be a crash every minute.

So there is a traffic cop in the middle. The cop holds up a hand and says "stop." The cop waves and says "go." The cop times each direction so cars from the north get a turn, then cars from the east, then pedestrians, then cars from the south, and so on. The cop never stops working. The cop is always there. The cop never lets two streams of traffic crash into each other.

That is the kernel. The cars are programs all wanting to use shared resources. The kernel decides when each one gets a turn. The kernel never sleeps (well, not really — when nothing is happening the kernel does sleep a tiny bit to save battery, but it wakes up the instant somebody needs it).

### Why so many pictures?

You might wonder why we have a factory picture, a restaurant picture, a librarian picture, and a traffic cop picture. The answer is: nobody can see the kernel. The kernel is invisible. So we have to use pictures to imagine what it is doing. Different pictures help with different ideas.

The **factory** picture is best for understanding that the kernel manages workers (processes).

The **restaurant** picture is best for understanding that the kernel stands between programs and dangerous hardware.

The **librarian** picture is best for understanding that the kernel keeps track of where every piece of data lives.

The **traffic cop** picture is best for understanding that the kernel takes turns and prevents crashes.

If one picture is not clicking, switch to another. Whichever one feels right is the one you should keep in your head.

### The five big jobs of the kernel

The kernel has five really important jobs. We are going to go through each one. The factory and restaurant pictures will keep showing up because they are the easiest way to remember everything.

#### Job 1: Sharing the brain

Your computer has a brain called the CPU. CPU stands for "Central Processing Unit," which is a fancy way of saying "the part that thinks." The CPU is what does all the actual work in a computer. When you click a button, the CPU figures out what should happen. When you watch a video, the CPU is doing the math to show each picture on the screen.

There is usually only one CPU in a computer. Sometimes there are a few (called "cores"), but it doesn't matter for now: the point is there are way fewer brains than there are programs that want to think.

So the kernel takes turns. The kernel says, "Web browser, you can think for a tiny moment. Done? Great, now music player, you can think. Done? Now text editor, your turn." It does this so fast that you can't even tell it's happening. To you it looks like every program is running at the same time. But really, the kernel is just super fast at switching between them.

It is like a teacher in a classroom calling on students. "Sarah, what's two plus two? Good. James, what's three plus three? Good. Sarah, what's four plus four? Good." If the teacher does this fast enough, both kids feel like the teacher is talking to them all the time, even though the teacher is only talking to one kid at a time.

The kernel piece that does this is called the **scheduler.** It schedules whose turn it is. Modern CPUs can switch between programs **billions of times per second.** That is faster than you can blink. That is faster than the lightning that comes out of clouds. That is so fast that you and I might as well just say "everything is running at the same time" because we can't see the gaps.

To make this really concrete: imagine you blink your eyes once. That blink takes about a third of a second. In that third of a second, the kernel might have switched between programs **a hundred million times.** A hundred million little turns. That's how blurry-fast the scheduler is. By the time you finish blinking, every program on your computer has had thousands of turns to think.

There are also tricks the kernel can do to be smarter. Some programs are more important than others. The music player should never get paused for too long, otherwise you hear a glitch. The clock in the corner doesn't need much CPU time. The kernel can give different programs different amounts of "turn time" to keep things smooth. This is sometimes called **priority.** Programs with higher priority get more turns. Programs with lower priority get fewer turns. You can ask the kernel to change a program's priority with the `nice` and `renice` commands, but most of the time you let the kernel decide.

Some programs are sleeping. They are not asking for a turn at all. They are just waiting for something to happen, like a key press or a network message. The kernel does not waste any time on sleeping programs. As soon as the thing they were waiting for happens, the kernel wakes them up, hands them a turn, and they get back to work.

#### Job 2: Sharing the desks (memory)

Memory is called RAM. RAM stands for "Random Access Memory," which is a fancy way of saying "scratch paper your computer can write on really fast." Every program needs RAM to do its work. The web browser needs RAM to remember what website you're on. The music player needs RAM to remember what song is playing. A game needs RAM to remember where you are on the level.

Picture a classroom. Each kid has a desk. The desk is where the kid does their homework. The kernel makes sure every program gets a desk, and the kernel makes sure no kid can lean over and erase another kid's homework.

This is really important. If one program could read another program's memory, that would be terrible. Imagine if your web browser could read what your password manager is holding. Or if a game could read what your bank app is holding. Or if a virus could read everything everyone is doing. So the kernel puts up invisible walls around every program's memory. We call these walls **memory protection.**

If a program tries to peek over the wall, the kernel slaps its hand and shuts it down. The program gets killed, the bad thing doesn't happen, and you usually see a little message like "Application crashed." That crash message is actually the kernel doing its job correctly. The kernel said "no" to a program that tried to do something bad.

**What if there aren't enough desks?**

Sometimes too many programs ask for too much memory. There aren't enough desks for everyone. The kernel has a clever trick. It takes some of the stuff sitting on a desk that nobody is currently using, and it carries that stuff out to the hallway and puts it on a temporary table out there. Now the desk is free for somebody else. Later, when the original program needs its stuff back, the kernel brings it back from the hallway.

The hallway is part of the hard drive. We call this the **swap space**, or just "swap." Using swap is way slower than using a real desk because the hallway is far away. Hard drives are slow. RAM is fast. So when your computer starts using a lot of swap, it gets sluggish. That is why people pay extra for more RAM. More desks means less running to the hallway.

Imagine the difference. A desk in the room is right there: stretching out your arm takes maybe one second. Walking to the hallway is much further: ten seconds out, ten seconds back. Twenty times slower! That is roughly what it feels like to the kernel when it has to use swap instead of RAM. Real numbers: RAM is about a hundred nanoseconds away. Swap on a fast SSD is about a hundred microseconds away. Swap on an old spinning hard drive is about ten milliseconds away. Each step up that ladder is roughly a thousand times slower than the previous one.

When you hear somebody say "my computer is thrashing," that means the kernel is constantly running to the hallway and back, never able to settle. The whole computer feels frozen because the CPU is just waiting on the hard drive most of the time. Adding more RAM is the cure. With more desks, the kernel almost never needs the hallway.

#### Job 3: Talking to all the gadgets

Your computer is connected to a ton of stuff. The keyboard. The mouse. The screen. The speakers. Maybe a printer. Maybe a microphone. Maybe a webcam. Maybe a USB stick. Maybe a Wi-Fi card. Each one of these gadgets speaks its own weird language. The keyboard sends little signals when keys go down and up. The mouse sends signals about how it moved. The Wi-Fi card sends bursts of radio waves.

The kernel has a special program for each gadget. We call these programs **drivers.** A driver is a translator. There is one driver for the keyboard. There is one driver for the mouse. There is one driver for the screen. When your web browser wants to show a picture, it doesn't know how the screen works. It just says, "Hey kernel, show this picture." The kernel hands the picture to the screen driver. The screen driver translates the picture into the exact electric signals the screen needs. The picture appears.

The Linux kernel comes with **over 10,000 drivers built in.** That is why people say Linux runs on "almost everything." Almost any gadget you can think of, somebody has written a driver for it and put it in the Linux kernel. Tiny computers the size of a credit card. Big supercomputers the size of a building. Phones. TVs. Fridges. Cars. Spaceships. They all have a Linux kernel inside, and they all work because Linux already has a driver for the gadget.

#### Job 4: Filing cabinets (file systems)

When you save a picture, it has to go somewhere. The somewhere is the hard drive. But the hard drive doesn't really understand "files" or "folders." A hard drive is just a giant block of tiny little storage spots. Imagine a wall with billions of tiny mailboxes, none of them labeled.

The kernel turns that wall of unlabeled mailboxes into a system that humans can understand. It builds folders. It builds files. It writes down which mailboxes hold which file. When you click "Save" on a picture, the kernel finds enough empty mailboxes, drops the picture into them, and writes down on a little card "this picture is in mailboxes 42, 43, 44, and 45." Later, when you click on the picture to open it, the kernel reads its little card, walks to those mailboxes, gets the data, and hands it to your photo viewer.

We call this whole organization a **file system.** A file system is the kernel's filing cabinet. It is what makes your hard drive feel like folders and files instead of a wall of mailboxes.

#### Job 5: Being the security guard

Finally, the kernel is the security guard. It decides who can do what. Most programs are not allowed to do scary things. Most programs cannot delete the files in the system folder. Most programs cannot install other programs. Most programs cannot read each other's secrets.

There is one special user on every Linux computer called **root.** The root user is like the principal of a school. The principal has a master key that opens every door. Most users on a computer are like students. They can only open their own locker. If a regular user tries to delete an important file, the kernel says "no, you don't have permission." If the root user tries the same thing, the kernel says "okay, you're the principal, knock yourself out."

That is why people say "be careful with `sudo`." `sudo` is the magic word that says, "for this one command, please pretend I am the principal." It is great when you really need to do something powerful. It is dangerous when you don't, because the principal can break things that students cannot.

### Wait, isn't Linux the whole operating system?

Lots of people say "Linux" when they mean the whole operating system. Like, when you hear somebody say, "I run Linux on my laptop," they probably mean they have something like Ubuntu or Fedora or Mint installed.

But really, the word "Linux" only refers to the kernel. The factory manager. Just the manager, by themselves, isn't a factory. You also need the workers, the front desk, the lunchroom, the parking lot, all the tools, the building. All those other things together with the kernel make an **operating system.**

Different operating systems use the same Linux kernel but bundle it with different other software. Ubuntu is one bundle. Fedora is another. Arch Linux is another. NixOS is another. Each of these is called a **distribution**, or **distro** for short. They all share the same engine (the Linux kernel) but the steering wheel and seats and paint job are different.

Picture cars. The engine is the kernel. The whole car is the operating system. Two cars can have the same engine but look completely different from the outside. Same with Linux distros. Same kernel, very different look and feel.

### Two worlds: user space and kernel space

The kernel keeps two completely separate worlds inside your computer. They have boring grown-up names but the idea is simple.

**User space** is where your programs live. Your web browser. Your music player. Your text editor. Your games. The little blinking clock. They all live in user space. User space programs have limited powers. They cannot touch hardware directly. They cannot mess with each other.

**Kernel space** is where the kernel lives. Only the kernel lives there. Kernel space has full power over everything. The kernel can talk to every gadget directly. The kernel can read every byte of memory. The kernel can do anything.

There is a big invisible wall between user space and kernel space. The wall is enforced by the CPU itself, not just by the kernel. Even if a program tried to break into kernel space, the CPU would catch it and slap its hand. This is sometimes called **ring 0** (the inside ring, where the kernel lives) versus **ring 3** (the outside ring, where programs live). Kernel = ring 0. Programs = ring 3. The CPU has hardware that makes sure ring 3 cannot do ring 0 things without permission.

### How does a program ask the kernel for help?

When a program needs something from the kernel, it raises its hand. The polite hand-raise is called a **system call**, often shortened to **syscall.**

Here are some everyday hand-raises:

- "Kernel, please open this file for me." — the syscall is `open()`.
- "Kernel, please read the next chunk of data from this file." — the syscall is `read()`.
- "Kernel, please connect me to this address on the internet." — the syscall is `connect()`.
- "Kernel, please start a new program." — the syscall is `execve()`.
- "Kernel, what time is it?" — the syscall is `clock_gettime()`.

There are over 400 of these in Linux. Every single thing your computer ever does eventually boils down to programs asking the kernel for help, the kernel doing it, and the kernel saying "here you go." Drawing a picture on the screen. Playing a sound. Sending a text message. Saving a file. Running a new program. They are all just polite syscalls under the hood.

### Processes: programs that are actually running

A **program** is a thing that exists. Like a recipe. Like a book sitting on a shelf. It is just data. It is not doing anything.

A **process** is a program that is actually running right now. Like the recipe in the middle of being cooked. Like the book in the middle of being read.

When you double-click an app, the kernel takes the program off the shelf and starts it as a process. The kernel gives it a desk (memory). The kernel gives it a turn at the brain (CPU time). The kernel gives it a badge number. The badge number is called a **PID**, which stands for "Process ID." Every running thing on your computer has a unique PID.

Each process thinks it has the whole computer to itself. The kernel keeps up this illusion for every single process at the same time. It is like a magician doing the same trick for a thousand people in the audience, and every single audience member feels like the trick was just for them. That is what the kernel is doing all day.

Right now, your computer probably has hundreds of processes running. Even when you are not using your computer, there are tons of little background helpers doing tiny jobs. We will see them soon.

### What happens when you click an app icon

This is a fun thing to walk through, because every kernel topic shows up in it.

1. You click the icon. The mouse driver in the kernel notices the click and tells the desktop program where you clicked.
2. The desktop program looks up which program belongs to that icon. Let's say it's Firefox.
3. The desktop program asks the kernel to start Firefox. This is the `execve` syscall.
4. The kernel finds the Firefox program file on disk (file system at work).
5. The kernel makes a new process for Firefox: gives it a PID, sets up its own memory area (memory management at work), and puts it on the scheduler's list (scheduler at work).
6. The kernel loads the Firefox program into memory.
7. The scheduler eventually picks Firefox to run. Firefox starts thinking.
8. Firefox immediately starts making syscalls: open this config file, read this saved data, connect to the internet, etc.
9. For each syscall, the kernel checks permissions (security at work), does the work, and returns the answer.
10. Firefox draws its window. It does this through more syscalls that go to the screen driver (drivers at work).
11. You see the window appear.

That whole sequence happens in maybe a second or two. Every single one of the kernel's five jobs got used. Now multiply that by every app you have ever opened in your life.

### A timeline of the Linux kernel

Just to give you a sense of scale, here is how the Linux kernel has grown over the years.

```
1991: Linus Torvalds posts kernel 0.01 from his bedroom in Finland.
      ~10,000 lines of code. Runs on one type of computer (Intel 386).
      No graphical interface. No networking. Almost no drivers.

1994: Kernel 1.0 ships. ~175,000 lines.
      First "real" release. People start using it on home computers.

1996: Kernel 2.0. Multi-CPU support. ~700,000 lines.

1999: Kernel 2.2. Better memory management. ~1.8 million lines.

2003: Kernel 2.6. Massive overhaul. New scheduler. ~5.2 million lines.
      This version powered most of the 2000s.

2011: Kernel 3.0. ~14 million lines.

2015: Kernel 4.0. Live patching support. ~19 million lines.

2019: Kernel 5.0. ~26 million lines.

2022: Kernel 6.0. ~30 million lines.

2026: Kernel 6.x is current. ~32+ million lines.
```

The kernel keeps growing. Most of the growth is in drivers (because new hardware keeps coming out) and new features. Some old code does get removed or rewritten, so it's not pure addition. But the trend is up and to the right.

### Why is the Linux kernel a big deal?

A college student named **Linus Torvalds** started writing the Linux kernel in **1991.** He just wanted a thing for his own computer. He shared it on the internet. Other people liked it. Other people started helping. Today, **over 20,000 different people** from thousands of companies have written code for the Linux kernel. It is the biggest group project in the history of the world.

It is **open source.** That means anyone can look at the code. Anyone can suggest changes. Anyone can use it for free. You don't have to buy it. You don't have to ask permission. You can grab a copy right now if you want.

Today, Linux runs on more devices than any other kernel. Android phones use Linux. Smart TVs use Linux. Wi-Fi routers use Linux. Cars use Linux. The space station uses Linux. **Over 90% of the servers that run the internet use Linux.** Whenever you watch a video online, or search for something, or send a message to a friend, there is a Linux kernel doing the work behind the scenes.

That is why this stuff is worth learning. Once you understand the kernel, you understand the thing that is running most of the computers on Earth.

## Concepts in Detail

### Why the word "kernel"?

The word "kernel" originally means the inside, the central part, of something. Like the kernel of a nut: the part inside the shell. The part that matters. The part you actually eat.

The operating system kernel is the inside, central part of the operating system. The part that matters. Everything else can be removed, and the system will still kind of work. But take the kernel out and there is nothing left.

This is why we say "kernel." It is the heart, the center, the real thing.

### What is a kernel?

A kernel is the part of an operating system that has full control over everything in the computer. It is the only software that can talk to the hardware directly. Everything else (every program you run) has to go through the kernel.

If we draw it as a picture:

```
+---------------------------------------------------+
|              YOU (the human)                      |
+---------------------------------------------------+
                    |
                    | clicks, taps, keystrokes
                    v
+---------------------------------------------------+
|       PROGRAMS (browser, music, games...)         |   <- user space
+---------------------------------------------------+
                    |
                    | system calls (polite hand-raise)
                    v
+---------------------------------------------------+
|                    KERNEL                         |   <- kernel space
|       (scheduler, memory, drivers, files)         |
+---------------------------------------------------+
                    |
                    | direct signals
                    v
+---------------------------------------------------+
|       HARDWARE (CPU, RAM, disk, screen...)        |
+---------------------------------------------------+
```

You at the top. Programs in the middle. The kernel below them. Hardware at the bottom. The kernel is the only thing that gets to touch hardware. Everything else has to ask the kernel.

### User space vs. kernel space

These are the two worlds we mentioned. Here is the wall between them in picture form:

```
USER SPACE                 ||      KERNEL SPACE
                           ||
[browser]                  ||      [scheduler]
[music player]             ||      [memory manager]
[text editor]              ||      [keyboard driver]
[game]                     ||      [screen driver]
[clock]                    ||      [Wi-Fi driver]
                           ||      [file system]
limited powers             ||      full powers
cannot touch hardware      ||      can touch all hardware
                           ||
                  THE WALL ||  (enforced by the CPU)
                           ||
              "knock knock" -->
              syscall      -->
                           ||
                           <-- "here is what you asked for"
```

A program in user space cannot just walk through the wall. It has to make a system call, which is like knocking on the wall. The kernel checks who is knocking, decides if it should answer, does the work, and passes the answer back through the wall. The program cannot see kernel space. The program cannot see what the kernel is doing inside.

This wall is what keeps your computer safe. If user space and kernel space were not separated, any program could break the whole computer. Any virus could break the whole computer. Because they are separated, even when a program does something bad, it can usually only break itself, not the whole machine.

### CPU sharing (the scheduler)

The kernel piece that decides whose turn it is on the CPU is called the **scheduler.** Imagine the scheduler as a very fast teacher in front of a class:

```
            +------------------------+
            |       SCHEDULER        |
            |  "okay, your turn..."  |
            +-----------+------------+
                        |
        +---------------+---------------+
        |               |               |
        v               v               v
   [browser]       [music]         [text editor]
   "I want         "I want         "I want
    to think"       to think"       to think"
        ^               ^               ^
        |               |               |
        +-------+-------+-------+-------+
                |               |
                v               v
            +-----+         +-----+
            | CPU |         | CPU |
            +-----+         +-----+
```

Each program is constantly putting its hand up saying "I want to think." The scheduler picks one and lets it use the CPU for a tiny moment (called a **time slice**, usually a few milliseconds). Then it pauses that program, picks another one, gives it a time slice, and so on. The whole game just keeps going forever as long as your computer is on.

If your computer has more than one CPU core, the scheduler can hand out time slices on multiple cores at the same time. More cores = more programs really running at once.

### Memory walls

The kernel gives every process its own memory. The trick is that each process thinks it has all the memory in the computer to itself. It can ask "kernel, give me the byte at address 1000" and the kernel hands it back something. But what the process thinks is "address 1000" is actually a totally different real spot in real RAM. The kernel keeps a little map for every process showing how the process's pretend addresses match up to the real spots in RAM.

This is called **virtual memory.** Each process has a "virtual" view of memory. The kernel translates between virtual addresses (what the process sees) and physical addresses (the real spots in RAM).

Picture:

```
   PROCESS A's view              PROCESS B's view
   +------------------+          +------------------+
   | 0:                |         | 0:                |
   | 1: My Stuff       |         | 1: My Stuff       |
   | 2: My Stuff       |         | 2: My Stuff       |
   | 3:                |         | 3: My Stuff       |
   | 4:                |         | 4:                |
   +------------------+          +------------------+
            |                              |
            |  kernel translation table    |
            v                              v
                   REAL RAM
   +-------------------------------------------------+
   | A's stuff | B's stuff | A's stuff | B's stuff   |
   | spread across real RAM in any order             |
   +-------------------------------------------------+
```

Process A and Process B each see themselves at "address 1." But really their stuff is in different real spots, totally separated. They literally cannot see each other's data because they are looking at different real spots through the kernel's pretend window.

The hardware piece that helps the kernel do this translation is called the **MMU**, which stands for "Memory Management Unit." The MMU is part of the CPU. It does the address translation in real time so things don't slow down.

### Swap space

When all the desks (RAM) are full, the kernel can use the hard drive as extra desks. The hard drive area used for this is called **swap space**, or just "swap."

Picture:

```
   FAST DESKS (real RAM)          SLOW HALLWAY (swap on disk)
   +---------------------+        +-----------------------+
   | active stuff        |        | sleeping stuff        |
   | being used right    |  <-->  | parked here for now   |
   | now                 |        | because RAM was full  |
   +---------------------+        +-----------------------+
        super fast                      slow
```

Swap is a safety net. It keeps programs alive even when RAM runs out. But it is slow, because hard drives (especially old spinning ones) are way slower than RAM. If your computer is "swapping a lot" (using swap a lot), it will feel sluggish. People with slow computers often add more RAM, which means less swap, which means faster computer.

### File system as filing cabinet

A file system is how the kernel turns a wall of unlabeled storage spots into folders and files. Picture:

```
   WHAT YOU SEE:                  WHAT THE DISK IS:
   +-------------------+          +-------------------+
   | /home/me/        |           | block 1: ???       |
   |   pictures/      |           | block 2: ???       |
   |     dog.jpg      |           | block 3: ???       |
   |     cat.jpg      |           | block 4: ???       |
   |   notes.txt      |           | block 5: ???       |
   | /etc/            |           | block 6: ???       |
   |   passwords      |           | ...                |
   +-------------------+          +-------------------+
              ^                              ^
              |                              |
              |   the kernel translates      |
              +------------------------------+
```

The kernel keeps a giant index card system. Each file has an entry that says "this file lives in blocks 4, 7, 12, and 19." When you read the file, the kernel goes to those blocks and reads them in order. When you save a new file, the kernel finds empty blocks, writes the data there, and adds a new entry to the index.

Different file systems do this in different ways. The most common one on Linux is called **ext4.** Other ones include **xfs**, **btrfs**, and **f2fs**. They all do the same basic job (turn a disk into folders and files), they just do it with different tricks under the hood.

### Drivers as translators

Picture every gadget as somebody who only speaks one specific language:

```
                +---------+           +---------+
                | KEYBOARD|           |  SCREEN |
                |  speaks |           |  speaks |
                | "click" |           | "pixel" |
                +----+----+           +----+----+
                     ^                     ^
                     |                     |
                     |                     |
              +------+----+         +------+-----+
              | KEYBOARD  |         |   SCREEN   |
              |  DRIVER   |         |   DRIVER   |
              | (in       |         | (in        |
              |  kernel)  |         |  kernel)   |
              +------+----+         +------+-----+
                     |                     |
                     |  speaks "syscall"   |
                     v                     v
                  +-----------------------------+
                  |    THE REST OF THE KERNEL   |
                  +-----------------------------+
                     |
                     v
                  [program: "what did the user type?"]
```

The keyboard sends weird electric signals. The keyboard driver speaks both "weird electric signals" and "syscall." It catches the signals, translates them to "the user pressed the letter A," and hands that to the kernel. Programs ask the kernel "what did the user type?" and the kernel says "A."

Same the other way around. A program says "show this picture on the screen." The kernel hands it to the screen driver. The screen driver translates "show this picture" into the exact electric signals the screen needs to light up its pixels.

A driver is just a translator that lives inside the kernel.

### Boot story: how the kernel even gets there

When you press the power button, the computer doesn't immediately have a kernel. The kernel is just a file sitting on the hard drive. Something has to find it, load it into RAM, and start it.

Picture:

```
Step 1: POWER ON
        |
        v
Step 2: FIRMWARE wakes up the basic hardware
        (BIOS or UEFI - these are tiny programs in a chip)
        |
        v
Step 3: FIRMWARE finds the BOOT LOADER on disk
        (usually GRUB on Linux)
        |
        v
Step 4: BOOT LOADER finds the KERNEL FILE on disk
        (usually called vmlinuz-something in /boot)
        |
        v
Step 5: BOOT LOADER copies the KERNEL into RAM
        and tells the CPU to start running it
        |
        v
Step 6: KERNEL initializes itself
        - sets up memory tables
        - starts the scheduler
        - loads drivers
        - mounts the file system
        |
        v
Step 7: KERNEL starts the FIRST PROGRAM
        (PID 1, usually systemd)
        |
        v
Step 8: SYSTEMD starts everything else
        (login screen, services, your shell, etc.)
        |
        v
Step 9: YOU LOG IN
        (you finally get to use the computer)
```

Every Linux computer goes through this exact dance, every single boot. From power-on to login screen takes maybe 10-30 seconds on modern hardware, depending on how much stuff is being started.

If you ever break your computer to where it won't boot, knowing this chain is what helps you figure out which step broke. Did the firmware not find the boot loader? Did the boot loader not find the kernel? Did the kernel start but then panic? Did the kernel succeed but PID 1 didn't start? Each step has its own kind of error message.

### Interrupts: how the kernel hears the doorbell

The kernel can't be staring at every gadget all the time, waiting for something to happen. That would waste tons of CPU time. Instead, gadgets ring a doorbell when something happens.

The doorbell is called an **interrupt.** When you press a key, the keyboard sends an interrupt. When the network card receives a packet, it sends an interrupt. When the timer ticks, it sends an interrupt.

When an interrupt arrives, the CPU stops whatever it was doing, drops everything, and runs a special little kernel function called an **interrupt handler.** The handler does just enough to record what happened (e.g., "key A was pressed") and then returns. The CPU goes back to whatever program it was running.

Picture:

```
Programs are running...
[browser thinking]
[music player thinking]

   *** ding! interrupt from keyboard! ***

Everything pauses.
Kernel runs the keyboard interrupt handler:
  "Oh, the user pressed key A. Add it to the input queue."
Done. Took a few microseconds.

Programs resume.
[browser thinking again]
[music player thinking again]

Eventually some program asks "what did the user type?"
Kernel: "Key A."
```

Every interrupt has a number, called an **IRQ** (Interrupt ReQuest). Different gadgets get different IRQs. The keyboard has one. The mouse has another. The network card has another. The kernel has a table that says "if IRQ 12 happens, run this handler. If IRQ 14 happens, run that handler."

Without interrupts, the kernel would have to constantly poll every gadget asking "anything happen yet? anything happen yet?" That would waste so much time. Interrupts let gadgets be quiet until they actually have something to say.

### The security guard role

The kernel decides who is allowed to do what. It does this with the help of two ideas: **users** and **permissions.**

Every running process belongs to a user. Each user has a unique number called a **UID** (user ID). When a program tries to do something, the kernel checks "what user is this program running as, and is that user allowed to do this thing?"

Picture a building:

```
   +----------------------------+
   |        BUILDING            |
   +----------------------------+
   | Front office:              |
   |   * principal can enter    |
   |   * teachers can enter     |
   |   * students cannot enter  |
   |                            |
   | Classroom 1:               |
   |   * teacher 1 can enter    |
   |   * teacher 2 cannot       |
   |   * principal can enter    |
   |   * student in class 1 can |
   |                            |
   | Locker room:               |
   |   * each student can only  |
   |     open their own locker  |
   +----------------------------+
```

The kernel is the security guard at the doors. It checks who you are before letting you in. The principal (root) can go anywhere. Teachers (admin users) can go to lots of places but not all. Students (regular users) can only go to a few places.

This is called **permissions.** Every file and every action on a Linux computer has permissions saying who can do what.

## Hands-On

Time to actually try things. You will need a terminal. A terminal is a black window where you type commands. On most Linux computers you can open one with **Ctrl-Alt-T**, or look for an app called "Terminal."

For each command below, type the part after the `$` and press Enter. The lines without `$` are what the computer prints back. Your output might look slightly different (different version numbers, different file names) but the shape of it should be the same.

If a command does not work and you get the message **`command not found`**, that means your computer doesn't have that program. That is okay. Move on to the next one.

### Which kernel am I running?

The command `uname -r` tells you the version number of the kernel currently running on your computer. The `-r` part means "release version."

```
$ uname -r
6.5.0-25-generic
```

That number means "Linux kernel version 6.5.0, build number 25, the generic build." Yours will probably be a different version. As long as it is not blank, the command worked.

For more details about your kernel, use `uname -a`:

```
$ uname -a
Linux mybox 6.5.0-25-generic #25-Ubuntu SMP Wed Jan 17 17:50:26 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux
```

That output says: I am Linux, my hostname is `mybox`, my kernel version is `6.5.0-25-generic`, my CPU type is `x86_64` (a 64-bit Intel/AMD chip), and I'm a GNU/Linux system.

If you get this error, your `uname` command is missing, which is super rare:

```
$ uname -r
bash: uname: command not found
```

Almost certainly that won't happen. If it does, your computer is unusual. Move on.

### What is my CPU?

Type this:

```
$ cat /proc/cpuinfo | head -20
processor       : 0
vendor_id       : GenuineIntel
cpu family      : 6
model           : 142
model name      : Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
stepping        : 10
microcode       : 0xf4
cpu MHz         : 1800.000
cache size      : 8192 KB
physical id     : 0
siblings        : 8
core id         : 0
cpu cores       : 4
apic count      : 1
initial apic id : 0
fpu             : yes
fpu_exception   : yes
cpuid level     : 22
wp              : yes
flags           : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm constant_addressing
```

What did we just do? `cat` is a command that reads files and shows them. `/proc/cpuinfo` is a magic file the kernel makes up on the fly. It is not a real file on disk. It is the kernel's way of saying "here are the facts about your CPU." `| head -20` means "only show the first 20 lines, please." Without `head -20`, you would see a wall of text, especially if you have many cores.

The first line, `processor : 0`, is core number 0 (we count from zero). If you scroll down, you will see `processor : 1`, `processor : 2`, and so on, one block per core.

If you see this error:

```
$ cat /proc/cpuinfo | head -20
cat: /proc/cpuinfo: No such file or directory
```

That means you are not on Linux, or `/proc` is not mounted. On real Linux this should always work.

### How much memory do I have?

The command `free -h` shows you how much RAM and swap you have, and how much is used. The `-h` part means "human-readable" (i.e. show megabytes and gigabytes instead of raw bytes).

```
$ free -h
               total        used        free      shared  buff/cache   available
Mem:            15Gi       4.2Gi       1.1Gi       287Mi        10Gi        10Gi
Swap:          2.0Gi          0B       2.0Gi
```

That output says: I have 15 gibibytes (basically gigabytes) of RAM total. 4.2 gigs are in use by programs right now. 1.1 gigs are completely free and unused. 287 megs are shared between programs. 10 gigs are sitting in "buff/cache" which is the kernel's smart use of leftover RAM (it remembers files it just read in case you ask for them again — this is good, not wasted). And 10 gigs are "available" if you start a new program that asks for memory.

Swap: I have 2 gigabytes of swap space. None of it is in use right now (which means I have plenty of RAM and don't need the hallway).

If you see something like:

```
$ free -h
bash: free: command not found
```

That is rare on Linux but possible if you have a very stripped-down system. You can use `cat /proc/meminfo` instead, but that output is huge.

### How much disk space do I have?

The command `df -h` shows you all your disks and how full they are. `df` stands for "disk free." `-h` again means human-readable.

```
$ df -h
Filesystem      Size  Used Avail Use% Mounted on
tmpfs           1.6G  1.8M  1.6G   1% /run
/dev/nvme0n1p2  468G  142G  303G  32% /
tmpfs           7.7G   30M  7.7G   1% /dev/shm
tmpfs           5.0M  4.0K  5.0M   1% /run/lock
/dev/nvme0n1p1  511M  6.1M  505M   2% /boot/efi
tmpfs           1.6G  120K  1.6G   1% /run/user/1000
```

Look at the line for `/`. That is your main disk, the root of your file system. In this example, it is 468 gigabytes total, 142 used, 303 available, 32 percent full. The "Mounted on" column says where in the file system that disk shows up. `/` means "the top of the tree" and `/boot/efi` is a tiny separate area used at boot time.

Several of those entries say `tmpfs`. Those are not real disks. They are little RAM-backed file systems the kernel makes for temporary stuff. They go away when you turn the computer off.

### What is running right now?

The command `ps aux | head -5` lists running processes. `ps` is "process status." The `aux` part is three little flags asking for "all users, full info, including ones without a terminal." `| head -5` cuts it down to the first five lines so we don't drown.

```
$ ps aux | head -5
USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root           1  0.0  0.0 168388 13396 ?        Ss   Apr27   0:02 /sbin/init splash
root           2  0.0  0.0      0     0 ?        S    Apr27   0:00 [kthreadd]
root           3  0.0  0.0      0     0 ?        I<   Apr27   0:00 [rcu_gp]
root           4  0.0  0.0      0     0 ?        I<   Apr27   0:00 [rcu_par_gp]
```

The first row is the header. Then each row is one process.

- `USER` — who started it (user `root` here).
- `PID` — process ID. PID 1 is special: it is `init`, the very first process the kernel starts at boot. Every other process is a child of PID 1 (or a child of a child).
- `%CPU` — how much CPU it's using.
- `%MEM` — how much RAM it's using.
- `COMMAND` — the actual program. Names in `[brackets]` are kernel threads, which are little kernel helpers.

To see all processes you can scroll, but on a normal computer there are hundreds. Try this for fun:

```
$ ps aux | wc -l
347
```

That counts the lines. So I have 346 processes running plus one header line, total 347. Hundreds! All happening at once, all sharing the CPU thanks to the scheduler.

### What did the kernel just say?

The kernel keeps a log of important events. The command `dmesg` shows that log. Some computers require `sudo` for this; some allow regular users. Try without `sudo` first.

```
$ dmesg | head -20
[    0.000000] Linux version 6.5.0-25-generic (buildd@lcy02-amd64-031) (x86_64-linux-gnu-gcc-12 (Ubuntu 12.3.0-1ubuntu1~23.04) 12.3.0, GNU ld (GNU Binutils for Ubuntu) 2.40) #25-Ubuntu SMP PREEMPT_DYNAMIC Wed Jan 17 17:50:26 UTC 2024
[    0.000000] Command line: BOOT_IMAGE=/boot/vmlinuz-6.5.0-25-generic root=UUID=abc123 ro quiet splash
[    0.000000] KERNEL supported cpus:
[    0.000000]   Intel GenuineIntel
[    0.000000]   AMD AuthenticAMD
[    0.000000]   Hygon HygonGenuine
[    0.000000]   Centaur CentaurHauls
[    0.000000]   zhaoxin   Shanghai
[    0.000004] BIOS-provided physical RAM map:
[    0.000005] BIOS-e820: [mem 0x0000000000000000-0x000000000009ffff] usable
[    0.000007] BIOS-e820: [mem 0x00000000000a0000-0x00000000000fffff] reserved
[    0.000008] BIOS-e820: [mem 0x0000000000100000-0x000000007a7fcfff] usable
[    0.000009] BIOS-e820: [mem 0x0000000000000000-0x000000007a7fdfff] type 20
[    0.000010] BIOS-e820: [mem 0x0000000000130000-0x000000003fffffff] reserved
[    0.000022] ACPI: Early table reserved
[    0.000027] BIOS-e820: [mem 0x0000000000130000-0x000000003fffffff] reserved
[    0.000054] x86_64: Detected hardware platform: Intel Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
[    0.000122] CPU0 microcode level: 0xf4
[    0.000142] random.org: started early init
[    0.000200] memory: ext4-fs initialized
[    0.000254] kernel: detected USB controller (xhci-hcd)
```

The numbers in `[brackets]` at the start of each line are seconds since the kernel started. So `[    0.000000]` means "at the moment the kernel woke up" and `[    0.000254]` means "254 microseconds later."

If you see this error:

```
$ dmesg | head -20
dmesg: read kernel buffer failed: Operation not permitted
```

That means your computer is configured to require admin powers for `dmesg`. Try with `sudo`:

```
$ sudo dmesg | head -20
[sudo] password for me:
[    0.000000] Linux version 6.5.0-25-generic ...
```

It will ask for your password. Type it in (you won't see anything as you type, that's normal) and press Enter.

### What kernel modules are loaded?

A **kernel module** is a piece of the kernel that can be added or removed while the kernel is running. Modules are how Linux supports thousands of gadgets without making the kernel huge: you only load the modules you need.

The command `lsmod` lists currently loaded modules. `ls` is the list verb here. `mod` is short for module.

```
$ lsmod | head -10
Module                  Size  Used by
nls_iso8859_1          16384  1
intel_rapl_msr         20480  0
intel_rapl_common      36864  1 intel_rapl_msr
x86_pkg_temp_thermal   16384  0
intel_powerclamp       20480  0
coretemp               20480  0
kvm_intel             434176  0
kvm                  1208320  1 kvm_intel
irqbypass              16384  1 kvm
```

The first column is the module name. The second is its size in bytes. The third is how many things are using it (and the names of those things if any). For example, `kvm` is being used by 1 thing, and that thing is `kvm_intel`. So `kvm_intel` depends on `kvm`. If you tried to unload `kvm`, it would refuse, because `kvm_intel` is still using it.

If `lsmod` is missing on your system you will see:

```
$ lsmod
bash: lsmod: command not found
```

This is rare but sometimes happens on minimal containers. On a normal desktop or server it should work.

### What is my hostname?

The hostname is the name of your computer on the network. The kernel keeps track of it.

```
$ hostname
mybox
```

That is the short name. To see the full name with any domain attached:

```
$ hostname -f
mybox.localdomain
```

Or check the magic file:

```
$ cat /proc/sys/kernel/hostname
mybox
```

That is the same thing the kernel sees. The `hostname` command just reads that file.

### How long has my computer been on?

```
$ uptime
 14:32:11 up  3 days, 4:17,  1 user,  load average: 0.21, 0.42, 0.31
```

This says: it is currently 14:32:11 on the clock. The computer has been up for 3 days and 4 hours and 17 minutes. There is 1 user logged in. The load averages are 0.21, 0.42, and 0.31 for the last 1, 5, and 15 minutes.

You can also see this raw in `/proc`:

```
$ cat /proc/uptime
275837.42 1924512.67
```

The first number is seconds since boot. The second is total CPU idle time across all cores. So this computer has been on for 275837 seconds (about 3 days, 4 hours) and the cores have been idle for a total of 1.9 million seconds (because there are several cores adding up their idle time).

### What is my IP address?

The kernel tracks all the network interfaces too. On modern systems, use `ip addr`:

```
$ ip addr | head -20
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: wlp3s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default qlen 1000
    link/ether aa:bb:cc:dd:ee:ff brd ff:ff:ff:ff:ff:ff
    inet 192.168.1.42/24 brd 192.168.1.255 scope global dynamic noprefixroute wlp3s0
       valid_lft 84372sec preferred_lft 84372sec
    inet6 fe80::1234:5678:9abc:def0/64 scope link noprefixroute
       valid_lft forever preferred_lft forever
```

Your output will be different. The interesting numbers are the `inet` lines. `127.0.0.1` is the loopback (the computer talking to itself). `192.168.1.42` is this computer's address on the local network. The kernel manages all of this.

If `ip` is missing, try the older `ifconfig`:

```
$ ifconfig
bash: ifconfig: command not found
```

That happens because `ifconfig` is being phased out. Use `ip addr` instead.

### Bonus: see real syscalls

The command `strace` runs another program and prints every single syscall it makes. It is a very powerful learning tool. Try this safe example. We'll trace `echo hello`:

```
$ strace -c echo hello
hello
% time     seconds  usecs/call     calls    errors syscall
------ ----------- ----------- --------- --------- -------------------
  0.00    0.000000           0         3           read
  0.00    0.000000           0         3           write
  0.00    0.000000           0         5           close
  0.00    0.000000           0         8           mmap
  0.00    0.000000           0         4           mprotect
  0.00    0.000000           0         2           munmap
  0.00    0.000000           0         3           brk
  0.00    0.000000           0         4           pread64
  0.00    0.000000           0         1         1 access
  0.00    0.000000           0         1           execve
  0.00    0.000000           0         1           arch_prctl
  0.00    0.000000           0         2         1 openat
------ ----------- ----------- --------- --------- -------------------
100.00    0.000000           0        37         2 total
```

That `-c` flag means "count syscalls and summarize at the end." The output says: just to print the word "hello" to your screen, the program made 37 system calls total. Three reads. Three writes. Eight mmaps. And so on. Every single thing that program did was a polite request to the kernel.

If `strace` is not installed:

```
$ strace -c echo hello
bash: strace: command not found
```

Then you can install it. On Ubuntu/Debian: `sudo apt install strace`. On Fedora: `sudo dnf install strace`. On Arch: `sudo pacman -S strace`.

## Common Confusions

### "I thought Linux WAS the operating system?"

**The confusion:** People say "I run Linux" and you assume "Linux" is the whole thing.

**The fix:** The word "Linux" technically only means the kernel — the manager. The whole operating system also includes thousands of other programs, libraries, and tools. When somebody says they run Linux, they usually mean a distro like Ubuntu or Fedora, which is the Linux kernel plus a giant pile of other software.

Think of it this way: an engine is not a car. Linux is the engine. Ubuntu is the car.

### "Why does my computer have a kernel and a brain?"

**The confusion:** Both are described as "in charge."

**The fix:** They are different. The brain (CPU) is the hardware part that does the actual thinking and math. The kernel is the software part that tells the CPU which program's thinking to do right now. The CPU is the muscle. The kernel is the boss telling the muscle what to lift.

The CPU could not run a computer by itself. It needs the kernel to tell it what to do. The kernel could not run a computer by itself either. It needs the CPU to actually do the math.

### "Why can't I just talk to the screen myself?"

**The confusion:** Why all this kernel stuff? Why can't a program just write to the screen directly?

**The fix:** Because every other program would also try to. Imagine fifty programs all writing to the screen at exactly the same time. They would overwrite each other. The screen would flicker between fifty different things. The keyboard input would arrive at the wrong program. It would be unusable.

The kernel is the manager that says "okay browser, your turn for the screen now." Every program goes through the kernel so the kernel can take turns and prevent crashes.

### "Why do I need a password for `sudo`?"

**The confusion:** It's MY computer, why do I need a password to do stuff on it?

**The fix:** The password is not because the computer doesn't trust you. It is because the kernel is making sure that the program asking for `sudo` is really being run by you, the human, and not by some sneaky background program. By asking for the password, it confirms a human is in front of the keyboard before allowing super-powerful commands. This stops viruses and malicious websites from doing nasty things behind your back.

### "Programs run on the CPU directly, right?"

**The confusion:** When I run my program, doesn't it just go straight to the CPU?

**The fix:** No. The kernel decides when your program gets to use the CPU. The kernel pauses it. The kernel resumes it. The kernel switches to a different program in between. Your program might run for a few thousandths of a second, get paused, run again, get paused, and so on. From your program's point of view, time seems continuous. But really, the kernel is constantly cutting in.

### "If RAM is full, won't my computer just stop?"

**The confusion:** Memory full = computer dies, right?

**The fix:** Not always. The kernel has the swap trick. It moves stuff that nobody is currently using out to disk, freeing up RAM for stuff that is currently being used. The computer slows down (because disk is slow) but it keeps going. Only if both RAM and swap fill up does the kernel actually have to start refusing to give memory, and even then it will usually pick the worst-behaving program and kill it (called "OOM kill" — out of memory kill) to save the rest.

### "Is the kernel always running?"

**The confusion:** Is the kernel a program that runs, like Firefox?

**The fix:** Sort of, but not really. The kernel is loaded into memory at boot time and stays there until you shut down. It is not running all the time the way Firefox is. It only "runs" when something happens that needs its attention: a syscall from a program, a hardware event (like a key press), or the scheduler timer ticking. In between those, the kernel is dormant and the actual programs are using the CPU. The kernel only wakes up when it's needed.

This is actually how the kernel keeps things fast. If the kernel were running all the time, less time would be left for your programs. So it stays out of the way until it's needed.

### "What's the difference between root and admin?"

**The confusion:** I've heard both words.

**The fix:** They mean basically the same thing on Linux. "Root" is the specific name of the all-powerful user (UID 0). "Admin" is just the general English word for somebody with admin powers. On Linux, the all-powerful admin is called root. On Windows, it's called Administrator. On macOS, it's also called root. The idea is the same: one super-user account that can do anything, and you should only use it when you really need to.

### "If everything goes through the kernel, isn't that slow?"

**The confusion:** A polite syscall every time = slow, right?

**The fix:** Syscalls are super fast. A modern CPU can do millions of them per second. The kernel is also extremely well-optimized for this. The cost of each syscall is tiny compared to the actual work being done. Plus, the safety and order the kernel provides is worth way more than the tiny speed cost. Without it, the system wouldn't work at all.

### "Are all kernels Linux kernels?"

**The confusion:** Is "kernel" just another word for Linux?

**The fix:** No. Every operating system has a kernel. Windows has the **NT kernel.** macOS has the **XNU kernel** (a mix of Mach and BSD). FreeBSD has the FreeBSD kernel. Linux has the Linux kernel. The word "kernel" is the general term. Different operating systems have different kernels. Each kernel is its own piece of software, written by different people, with different design choices.

### "Why does my screen freeze sometimes if the kernel is so fast?"

**The confusion:** Hundreds of millions of switches per second, yet sometimes my mouse stops moving.

**The fix:** Almost always, when the screen freezes, the problem is not the kernel itself. The problem is that the kernel is waiting on something slow. Maybe the disk is busy. Maybe a program is in a tight loop and the scheduler is having a hard time pre-empting it. Maybe RAM is full and the kernel is swapping like crazy. The kernel is still running fine in the background, but its hands are tied because the slow part of the computer is being slow. When you free up the slow thing (kill the program, restart, free up memory), the kernel goes back to feeling fast again.

### "Why do I have to learn the command line?"

**The confusion:** I have a graphical desktop. Why type commands?

**The fix:** You don't *have* to. You can do most everyday things from the desktop. But the command line is where the kernel really lives. Every command you type in the terminal is a much more direct conversation with the kernel than a click on a button. By learning the command line, you're learning the language of the kernel itself, which means you can do things that no graphical app exposes. Plus, the command line is the same on every Linux machine in the world, while the desktop changes between distros.

### "Is the kernel the same as the operating system kernel I read about in school?"

**The confusion:** Sometimes textbooks say "kernel" and sometimes "OS kernel."

**The fix:** Same thing. The word kernel almost always refers to the operating system kernel. There are other technical uses of the word "kernel" in computer science (like in math or graphics), but in this context it always means the boss software in the operating system.

### "Can I write my own kernel?"

**The confusion:** Is the kernel something only super-experts make?

**The fix:** Anyone can write a kernel. People do it as a hobby. There are tutorials online. Of course, writing a kernel as featureful as Linux takes thousands of people thousands of years of work, but writing a tiny "hello world" kernel that boots and prints something is a popular weekend project for college students. You won't be doing it tomorrow, but it's not magic. It's just code, like any other code.

### "What does it mean when somebody says the kernel panicked?"

**The confusion:** A computer crash with the words KERNEL PANIC scrolling on the screen sounds scary.

**The fix:** A kernel panic happens when the kernel hits an error so bad it can't safely keep running. Rather than risk corrupting your data, the kernel deliberately stops everything and prints a message. It is the kernel saying "I have lost my mind, I cannot trust myself anymore, I am stopping before I do damage." Usually the only fix is to reboot. Modern Linux is very stable so panics are rare. When they happen, the kernel writes the panic message to a special log so you can read it after reboot.

### "Is /boot something I should ever touch?"

**The confusion:** I see a folder called `/boot` and wonder what's in it.

**The fix:** `/boot` contains the kernel files (`vmlinuz-*`), the initial RAM disk files (`initrd.img-*`), and the boot loader's config. Don't randomly delete files in `/boot`. If you remove the wrong file, your computer won't boot. The package manager on your distro takes care of `/boot` for you. Just leave it alone unless you know what you're doing.

### "Why does dmesg need sudo on my computer but not on my friend's?"

**The confusion:** Same command, different behavior.

**The fix:** Some distros set a kernel parameter called `kernel.dmesg_restrict` to 1, which makes `dmesg` require admin powers. Other distros leave it at 0. Nothing wrong with either. Both are valid security choices. To check which one you have:

```
$ cat /proc/sys/kernel/dmesg_restrict
1
```

If it's 1, you need sudo. If it's 0, anyone can read dmesg.

### "Is the file system the same as the hard drive?"

**The confusion:** People say "file system" and "drive" interchangeably.

**The fix:** Not quite. The hard drive is the physical hardware. The file system is the structure on top of it: the folders, files, indexes, and rules. You can have one drive with one file system. Or one drive split into many partitions, each with a different file system. Or multiple drives all combined into one file system. The hard drive is the box. The file system is the organization inside the box.

### "When I delete a file, does it really go away?"

**The confusion:** Delete = gone forever?

**The fix:** Not really. When you delete a file, the kernel just removes the entry from the index that points to its data blocks. The actual data is still on the disk until something else writes over it. That is why deleted files can sometimes be recovered with special tools, and why people who really want to erase data use commands like `shred` that overwrite the actual blocks before deleting them.

### "What is /tmp for?"

**The confusion:** I see a `/tmp` folder. What's it for?

**The fix:** `/tmp` is a temporary scratch space. Programs use it to store little files they need for a moment but don't care about long-term. On most Linux systems, `/tmp` is wiped at every reboot. Don't store anything important in `/tmp`. On many systems, `/tmp` is actually a tmpfs, meaning it lives in RAM and is super fast.

### "Why do programs need installing? Can't I just run them?"

**The confusion:** What does "install" even mean?

**The fix:** Installing a program means: copy the program files to standard places (so the kernel can find them), copy any libraries the program depends on, set up file permissions so it can run, register it in any system-wide indexes, and possibly set up shortcuts. If you skip installing, the program might still run if you point at it directly, but it won't be in standard places and might miss its libraries. The package manager (`apt`, `dnf`, `pacman`, `nix`) handles all of this for you.

### "Why does the kernel have so many versions?"

**The confusion:** I see kernel 4.x, 5.x, 6.x, hundreds of subversions. Why so many?

**The fix:** Linux is constantly being improved. Each version is a snapshot in time. The big numbers (5, 6, etc.) are major versions — they add big new things. The small numbers are minor versions — small improvements and fixes. The kernel team releases a new minor version every couple of months. You can pick whichever version you want. Most people just use whatever their distro picked for them, which is usually a stable one with security fixes backported.

### "If a program crashes, does it take the kernel down with it?"

**The confusion:** A crashing program seems pretty serious.

**The fix:** No, almost never. The whole point of the user space / kernel space split is that a misbehaving program can only hurt itself. The kernel is protected. So when Firefox crashes, only Firefox dies. Everything else (including the kernel and all your other programs) keeps running. The kernel might even print a polite message in the log saying "Firefox died, here's what it was doing when it crashed." This is one of the kernel's most important features: containment.

### "Where do kernel updates come from?"

**The confusion:** Who tells my computer to update its kernel?

**The fix:** Your distro maintainers. The Linux kernel project releases a new kernel. The folks who maintain your distro (Ubuntu, Fedora, etc.) test it, package it for their users, and push it through the package manager. When you run an update command (`apt update && apt upgrade`, `dnf update`, etc.), you might pull in a new kernel. Usually you have to reboot to actually start using the new kernel, because the running kernel is in memory and a reboot is the simplest way to swap it out.

### "If the kernel is software, what's the firmware then?"

**The confusion:** People throw the word "firmware" around too.

**The fix:** Firmware is software that lives on a tiny chip inside a piece of hardware. Your keyboard has firmware (a tiny program that runs the keyboard). Your hard drive has firmware. Your motherboard has firmware (the BIOS or UEFI). Firmware runs before the kernel even loads. Firmware is what lets the computer wake up at all. Once the firmware does its first job (find the kernel and start it), the kernel takes over.

You can think of it like: firmware is the startup helper. Kernel is the boss. Programs are the workers.

### Experiment 16: Read your computer's birth time

```
$ uptime -s
2026-04-23 18:04:11
```

That is the exact moment your computer was last turned on. Useful for figuring out "when did this computer last reboot?"

### Experiment 17: Find a file by name

```
$ find /etc -name "hostname" 2>/dev/null
/etc/hostname
```

The `find` command walks through folders and finds files matching what you ask for. The `2>/dev/null` part hides any "permission denied" error messages so the output stays clean. Really useful for "where is that file?"

### Experiment 18: Watch live process changes

```
$ watch -n 1 'ps aux --sort=-%cpu | head -5'
```

The `watch` command runs another command over and over. `-n 1` means "every 1 second." So this will show you the top 5 CPU users, refreshed every second. Press Ctrl-C to exit.

### Experiment 19: See how much disk a folder uses

```
$ du -sh /home/me 2>/dev/null
4.2G    /home/me
```

`du` is "disk usage." `-s` means "summary" (don't list each subfolder). `-h` is human-readable. So this says my home folder uses 4.2 gigabytes total.

### Experiment 20: Discover which version of every common tool you have

```
$ bash --version | head -1
GNU bash, version 5.2.15(1)-release (x86_64-pc-linux-gnu)

$ python3 --version
Python 3.11.6

$ gcc --version | head -1
gcc (Ubuntu 12.3.0-1ubuntu1~23.04) 12.3.0
```

Almost every command-line tool answers `--version`. Try it on whatever tool you are curious about.

## Vocabulary

| Term | Plain English |
|------|---------------|
| **Kernel** | The boss software that manages the computer. Always running once the computer starts. |
| **Operating system** | The kernel plus all the other software bundled with it. |
| **User space** | Where regular programs live. Limited powers. |
| **Kernel space** | Where the kernel lives. Full powers over hardware. |
| **Process** | A program that is currently running. Has its own memory and a PID. |
| **Program** | A file on disk that can be turned into a process when run. |
| **RAM** | Random Access Memory. Fast scratch paper for the computer. |
| **Swap** | Disk space used as extra slow memory when RAM fills up. |
| **CPU** | Central Processing Unit. The brain that does math and decisions. |
| **Core** | One brain inside a CPU. Modern CPUs have many cores. |
| **Scheduler** | The part of the kernel that decides which process runs on the CPU next. |
| **Driver** | A small kernel program that knows how to talk to one specific gadget. |
| **File system** | The kernel's filing cabinet on disk: turns blocks into folders and files. |
| **/proc** | A magic folder created by the kernel that gives info about the system and processes. |
| **/sys** | Another magic folder created by the kernel for hardware and driver info. |
| **Syscall** | A polite request from a program to the kernel. The only way to enter kernel space. |
| **Fork** | A syscall that makes a copy of the current process. New PID, same code. |
| **Exec** | A syscall that replaces the current process's program with a different one. |
| **PID** | Process ID. A unique number for every running process. |
| **Terminal** | A black window where you type commands. |
| **Shell** | The program inside the terminal that reads your commands and runs them. |
| **Bash** | The most common shell on Linux. Stands for "Bourne Again SHell." |
| **Root** | The all-powerful user account (UID 0) on Linux. The principal. |
| **Sudo** | A command that lets a regular user run one command as root, after typing a password. |
| **Distro** | Short for "distribution." A bundle of the Linux kernel plus other software. (Ubuntu, Fedora, etc.) |
| **GNU** | A project that wrote many of the basic tools on Linux. "GNU/Linux" means GNU tools plus Linux kernel. |
| **Linux** | Strictly: the kernel started by Linus Torvalds in 1991. Loosely: any GNU/Linux distro. |
| **Kernel module** | A piece of the kernel that can be loaded or unloaded while the kernel runs. |
| **lsmod** | A command that lists currently loaded kernel modules. |
| **modprobe** | A command that loads a kernel module (with all its dependencies). |
| **Page** | A small chunk of memory the kernel manages as one unit. Usually 4 kilobytes. |
| **Page table** | The kernel's translation chart from virtual addresses to real RAM addresses. |
| **Virtual memory** | The pretend memory each process sees, translated by the kernel. |
| **MMU** | Memory Management Unit. CPU hardware that helps translate virtual addresses to real ones. |
| **Ring 0** | The most-privileged CPU mode. Where the kernel runs. |
| **Ring 3** | The least-privileged CPU mode. Where regular programs run. |
| **Interrupt** | A signal from hardware that says "hey, look at me right now." Pauses the CPU. |
| **IRQ** | Interrupt ReQuest. A specific number used by hardware to signal an interrupt. |
| **dmesg** | A command that shows the kernel's startup and event log. |
| **journalctl** | A command that shows the system journal (a more modern log on systemd systems). |
| **systemd** | A common init system on Linux. The first user-space program started by the kernel. |
| **Init** | The first user-space program started after the kernel. PID 1. |
| **Boot** | The whole process of turning a computer on and getting to a usable state. |
| **GRUB** | A common boot loader on Linux. Loads the kernel into memory. |
| **UEFI** | Modern firmware (replaces BIOS). Wakes up the computer and starts the boot loader. |
| **BIOS** | Old-school firmware. Same job as UEFI but older. |
| **Firmware** | Tiny software baked into hardware chips. Runs before the kernel. |
| **Memory protection** | Kernel feature that stops one process from reading another's memory. |
| **OOM kill** | "Out Of Memory" kill: when memory runs out completely, the kernel kills the worst process. |
| **tmpfs** | A file system that lives in RAM, not on disk. Goes away when the computer turns off. |
| **ext4** | The most common Linux file system. |
| **strace** | A tool that prints every syscall a program makes. Great for learning. |
| **uname** | A command that prints kernel name, version, and CPU architecture info. |
| **`free`** | A command that shows RAM and swap usage. |
| **`df`** | A command that shows disk usage. |
| **`ps`** | A command that lists running processes. |
| **UID** | User ID. A unique number for each user account. Root is UID 0. |
| **Permissions** | Rules saying who can read, write, or run each file. |
| **Open source** | Source code anyone can read, copy, and change. The Linux kernel is open source. |
| **Linus Torvalds** | The Finnish college student who started Linux in 1991. Still leads it today. |
| **Hardware** | The physical parts of a computer. |
| **Software** | Programs and data that run on hardware. |

## Try This

Here are some safe experiments. None of these will break anything. They will teach you to feel comfortable poking at the kernel.

### Experiment 1: Watch processes change in real time

Run `top` (just type `top` and press Enter). It is like Task Manager. You will see a constantly-updating list of processes. Watch the CPU column. The processes at the top are using the most CPU right now.

To leave `top`, press the letter `q`.

### Experiment 2: Find out exactly what kernel build your system has

Try all of these:

```
$ uname -r
$ uname -a
$ cat /proc/version
```

The third one is fun because `/proc/version` is another magic file the kernel creates to describe itself.

### Experiment 3: Count how many CPU cores you have

```
$ nproc
8
```

That's the count. Or, the long way:

```
$ grep -c ^processor /proc/cpuinfo
8
```

That counts how many `processor` lines are in `/proc/cpuinfo`. Should match `nproc`.

### Experiment 4: See how busy your CPU is

```
$ uptime
 14:32:11 up  3:47,  2 users,  load average: 0.45, 0.62, 0.71
```

The "load average" numbers are for the last 1, 5, and 15 minutes. They roughly tell you how busy your CPU has been. A number near or below the number of cores means "not very busy." Way above the number of cores means "overloaded."

### Experiment 5: Check the size of one specific kernel module

```
$ lsmod | grep usb
usbhid                 65536  0
usbcore               327680  4 usbhid,xhci_hcd,xhci_pci
usb_common             16384  3 usbhid,usbcore,xhci_pci
```

That `grep usb` filters the list to only show modules with "usb" in the name. The numbers are how big the module is in memory.

### Experiment 6: Find your own PID

When you are typing commands in a terminal, the shell itself is a process. To see its PID:

```
$ echo $$
12345
```

That special variable `$$` means "the PID of the current shell." Yours will be different.

### Experiment 7: Make a tiny file and trace it

If you have `strace` installed, try this:

```
$ strace -e trace=openat,write -o /tmp/trace.txt echo hello > /tmp/hello.txt
$ cat /tmp/trace.txt
openat(AT_FDCWD, "/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3
openat(AT_FDCWD, "/lib/x86_64-linux-gnu/libc.so.6", O_RDONLY|O_CLOEXEC) = 3
write(1, "hello\n", 6) = 6
+++ exited with 0 +++
```

That's the kernel's view of what happened when you ran `echo hello`. It opened a couple of system files (the C library), then made a single `write` syscall to send the word "hello\n" out, then exited. Six bytes written, six bytes returned. No magic. Just a polite syscall.

### Experiment 8: List the magic /proc files

```
$ ls /proc | head -30
1
10
1000
1042
107
12345
buddyinfo
cgroups
cmdline
consoles
cpuinfo
crypto
devices
diskstats
dma
driver
execdomains
filesystems
fb
fs
interrupts
iomem
ioports
key-users
keys
kmsg
kpagecount
kpageflags
loadavg
locks
```

Notice all those numbers? Each number is a PID. There is one folder per running process. Inside each folder is information about that process.

```
$ cat /proc/1/comm
systemd
```

That tells you the name of process 1 (which is the init process). It is `systemd` on most modern distros.

### Experiment 9: See how many open files there are

```
$ cat /proc/sys/fs/file-nr
3520     0       9223372036854775807
```

The first number is how many files are open right now across the whole system. The second is "free" file slots that were used and released. The third is the maximum allowed. So this system has 3520 open files at this exact moment.

### Experiment 10: Watch the kernel boot in slow motion

If you reboot your computer (don't do this if you have unsaved work), at the GRUB menu you can press `e` to edit the boot parameters, remove the words `quiet splash`, and then press Ctrl-X to boot. The kernel's startup messages will scroll across the screen instead of being hidden by a splash logo. You can read them as the system boots and see all the things the kernel does on the way up.

This experiment is safe (the changes are temporary, just for that one boot), but only do it when you have the time and don't have important work open.

### Experiment 11: Find the biggest memory hogs

```
$ ps aux --sort=-%mem | head -6
USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
me          3127  4.2 12.3 4517284 1934920 ?     Sl   09:21  12:34 /usr/lib/firefox/firefox
me          4112  1.1  4.5 1234568  712304 ?     Sl   09:25   3:11 /usr/bin/code
me          3245  0.4  2.1  587432  341288 ?     Sl   09:21   1:04 /usr/bin/spotify
me          5023  0.3  1.9  478990  298412 ?     Sl   10:00   0:42 /usr/bin/slack
root        1234  0.0  0.8  167234  128342 ?     Ss   Apr27   0:11 /usr/lib/systemd/systemd
```

The `--sort=-%mem` flag sorts by memory usage, biggest first. The `-` (minus) means "biggest first." Without it you'd get smallest first. This is a great way to find which program is eating up your RAM.

### Experiment 12: Find the biggest CPU hogs

Same idea but with CPU:

```
$ ps aux --sort=-%cpu | head -6
USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
me          3127 14.2 12.3 4517284 1934920 ?     Sl   09:21  42:11 /usr/lib/firefox/firefox
me          7234  9.1  3.2  712384  502311 ?     Sl   09:35  18:24 /usr/bin/some-game
me          4112  1.1  4.5 1234568  712304 ?     Sl   09:25   3:11 /usr/bin/code
me          3245  0.4  2.1  587432  341288 ?     Sl   09:21   1:04 /usr/bin/spotify
root        1234  0.0  0.8  167234  128342 ?     Ss   Apr27   0:11 /usr/lib/systemd/systemd
```

Now you know who's hogging the brain.

### Experiment 13: Read what your kernel was started with

```
$ cat /proc/cmdline
BOOT_IMAGE=/boot/vmlinuz-6.5.0-25-generic root=UUID=abc123-def456 ro quiet splash
```

That is exactly the command line the boot loader (GRUB) used to start the kernel. `BOOT_IMAGE` is which kernel file. `root=UUID=...` tells the kernel which disk to use as `/`. `ro` means start read-only. `quiet splash` hides the kernel messages and shows a splash screen at boot.

### Experiment 14: Check kernel parameters live

The kernel exposes thousands of tweakable settings under `/proc/sys`. Most you should not touch, but reading them is harmless.

```
$ cat /proc/sys/kernel/ostype
Linux

$ cat /proc/sys/kernel/osrelease
6.5.0-25-generic

$ cat /proc/sys/kernel/version
#25-Ubuntu SMP PREEMPT_DYNAMIC Wed Jan 17 17:50:26 UTC 2024

$ cat /proc/sys/kernel/hostname
mybox
```

Each of these is just reading a different piece of kernel state. None of them harm anything.

### Experiment 15: Watch kernel logs scroll

```
$ sudo dmesg -w
```

The `-w` flag means "watch": show the existing log and then keep showing new lines as they arrive. Plug in a USB stick. Watch new lines appear about the kernel detecting it. Press Ctrl-C to exit.

## A Few More Big-Picture Notes

### Stable does not mean static

You might hear that Linux is "stable." That word can be confusing. Stable does not mean nothing changes. It means the kernel is reliable: it doesn't crash, it doesn't randomly lose data, it does what it is supposed to do. The Linux kernel actually changes constantly. A new version comes out roughly every 9-10 weeks. Each new version has hundreds or thousands of changes. New drivers get added. Bugs get fixed. New features get included. You can stay on an older version forever if you want, or you can always be on the bleeding-edge latest.

There are two main kinds of releases:

- **Mainline releases** — the latest, with all the newest stuff. Released every couple of months.
- **LTS (Long Term Support) releases** — picked once a year or two, kept patched for years. Used by most distros that prioritize stability.

This is why the same Linux kernel powers both your shiny new laptop and a five-year-old server in a closet somewhere. Same family, different versions.

### The kernel is small (but not really)

People sometimes say "Linux is small." Compared to a whole operating system, the kernel is small. It is the central core, after all. But the actual size of the Linux kernel source code is about **30 million lines** as of 2024. That's not small at all in absolute terms. It's just small compared to the total amount of software that makes up a complete distro, which includes libraries, desktop, applications, documentation, fonts, sounds, and tons more.

Most of those 30 million lines are device drivers. The actual core of the kernel (scheduler, memory manager, file systems) is maybe a million lines. Drivers make up the rest because Linux supports thousands of different hardware devices and each one needs its own driver.

### You don't normally see the kernel doing anything

In normal computer use, you never see the kernel. You see programs. You see the desktop. You see the windows. The kernel is invisible by design. It is doing its job perfectly when you don't notice it at all.

That is part of why the kernel is hard to understand at first. With a regular program you can see it on the screen. With the kernel, all you can see are its effects. This sheet has tried to give you ways to peek at the kernel through `/proc`, through `dmesg`, through `strace`, through `ps`. Use those tools whenever you want to feel like the kernel is real instead of just a hidden manager.

### Why does this matter to me, the reader?

Because once you understand the kernel a little bit, every weird thing your computer does suddenly has a story. "Why is my computer slow?" — maybe it's swapping. "Why did my program crash?" — maybe the kernel killed it for trying to access memory it shouldn't have. "Why won't this USB stick work?" — maybe there's no driver for it. "Why does my computer take so long to boot?" — maybe it's loading a lot of kernel modules.

You don't need to be a kernel developer. You just need to know enough that the words make sense and you can take a guess at what's going on. That is what this sheet is for.

## Where to Go Next

Once this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal:

- **`cs fundamentals linux-kernel-internals`** — the dense reference. The real names of every data structure, every kernel subsystem, every thing.
- **`cs detail fundamentals/linux-kernel-internals`** — the academic underpinning. Math, formal semantics, complexity analysis.
- **`cs kernel-tuning sysctl`**, **`cs kernel-tuning cgroups`**, **`cs kernel-tuning namespaces`**, **`cs kernel-tuning ebpf`** — applied tuning sheets.
- **`cs system strace`** — see syscalls live for any program. The single most important learning tool for the kernel from the user-space side.
- **`cs system gdb`** — the debugger. Once you get into the weeds, you'll want this.
- **`cs performance perf`**, **`cs performance bpftrace`** — observability and tracing.
- **`cs fundamentals how-computers-work`** — backs all the way up to "what is a CPU, what is RAM" if you skipped that.
- **`cs fundamentals ebpf-bytecode`** — the in-kernel safe VM.
- **`cs containers docker`** — what containers actually are (namespaces + cgroups + capabilities).

## See Also

- `fundamentals/linux-kernel-internals` — engineer-grade reference.
- `fundamentals/how-computers-work` — basic computer hardware story.
- `fundamentals/ebpf-bytecode` — the in-kernel safe VM.
- `system/strace` — trace every syscall a program makes.
- `system/gdb` — interactive debugger for processes.
- `system/systemd` — the init system that PID 1 actually runs.
- `kernel-tuning/sysctl` — kernel runtime parameter tuning.
- `kernel-tuning/cgroups` — resource limits.
- `kernel-tuning/namespaces` — process isolation primitives.
- `kernel-tuning/ebpf` — applied eBPF.
- `containers/docker` — namespaces + cgroups + capabilities, packaged.
- `performance/perf` — sampling profiler for the kernel.
- `performance/bpftrace` — DTrace-style tracing for Linux.

## References

- **kernel.org** — the official Linux kernel website. Has source code, change logs, and the formal docs.
- **"The Linux Kernel"** by Rusty Russell — old but still beloved beginner book about how Linux works inside.
- **"Linux Kernel Development"** by Robert Love — slightly deeper, very approachable. Pick it up once this sheet feels easy.
- **`man 7 capabilities`** — manual page about the fine-grained permission system Linux uses.
- **`man 5 proc`** — full manual page for `/proc`. Type `man 5 proc` in your terminal to read it without leaving the terminal.
- **`man 5 sysfs`** — full manual page for `/sys`. Same idea: `man 5 sysfs` in the terminal.
- **`man 2 syscalls`** — list of every syscall in your kernel with one-line descriptions. Great for browsing.
- **The Linux Programming Interface** by Michael Kerrisk — the encyclopedia of Linux from the user-space side. Save for later.
- **`info coreutils`** — info pages for the GNU coreutils (`ls`, `cat`, `df`, `free`, etc.). Read with `info coreutils`.

Tip: every reference above can be read inside your terminal. Most are accessible via `man` or `info`. The book references can be downloaded as PDFs and read in a terminal-based PDF viewer like `zathura` or just opened with `less` if you grab the plain-text version. You really do not need to leave the terminal.

— End of ELI5 —

When this sheet feels boring (and it will, faster than you think), graduate to `cs fundamentals linux-kernel-internals` — the engineer-grade reference. It uses real names for everything: data structures, subsystems, syscalls, all of it. After that, `cs detail fundamentals/linux-kernel-internals` gives you the academic underpinning. By the time you've read both, you will be reading kernel source code without a flinch.

### One last thing before you go

Pick one command from the Hands-On section that you have not run yet. Run it right now. Read the output. Try to figure out what each part means, using the Vocabulary table as your dictionary. Don't just trust this sheet — see for yourself. The kernel is a real thing. It is on your computer, doing its job, right now. The commands in this sheet let you peek at it.

Reading is good. Doing is better. Type the commands. Watch the kernel respond.

You are now officially started on your kernel journey. Welcome.

The whole point of the North Star for the `cs` tool is: never leave the terminal to learn this stuff. Everything you need is here, or one `man` page away, or one `info` page away. There is no Google search you need to do to start understanding the Linux kernel. You can sit at your terminal, type, watch, read, and learn forever.

Have fun. The kernel is happy to be poked at. Nothing on this sheet will break anything. Try things. Type commands. Read what comes back. The more you do, the more it all clicks into place.

— End of ELI5 — (really this time!)
